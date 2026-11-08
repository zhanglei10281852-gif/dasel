package json

import (
	"bytes"
	"errors"
	"fmt"
	json "github.com/goccy/go-json"
	"github.com/tomwright/dasel/v3/model"
	"github.com/tomwright/dasel/v3/parsing"
	"io"
	"strings"
)

// ErrJSONMaxDepthExceeded is returned when JSON nesting depth exceeds maxJSONDepth.
var ErrJSONMaxDepthExceeded = errors.New("json nesting depth exceeded")

const maxJSONDepth = 10_000

var _ parsing.Reader = (*jsonReader)(nil)

func newJSONReader(options parsing.ReaderOptions) (parsing.Reader, error) {
	return &jsonReader{}, nil
}

type jsonReader struct{}

// Read reads a value from a byte slice.
// When the input contains multiple JSON values (NDJSON), they are returned
// as a branch-marked slice, mirroring the YAML multi-document behaviour.
func (j *jsonReader) Read(data []byte) (*model.Value, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	var results []*model.Value

	for decoder.More() {
		v, err := j.decodeValue(decoder, 0)
		if err != nil {
			return nil, err
		}
		results = append(results, v)
	}

	switch len(results) {
	case 0:
		return model.NewNullValue(), nil
	case 1:
		return results[0], nil
	default:
		slice := model.NewSliceValue()
		slice.MarkAsBranch()
		for _, v := range results {
			if err := slice.Append(v); err != nil {
				return nil, err
			}
		}
		return slice, nil
	}
}

// NewStreamReader creates a StreamReader that decodes consecutive top-level
// JSON values from r, one value at a time.
//
// A single value may span multiple reads or multiple lines, and arrays,
// objects and scalars are each treated as one value.
func NewStreamReader(r io.Reader) *StreamReader {
	tracked := &errorTrackingReader{r: r}
	decoder := json.NewDecoder(tracked)
	decoder.UseNumber()
	return &StreamReader{
		decoder: decoder,
		reader:  &jsonReader{},
		r:       tracked,
	}
}

// StreamReader reads top-level JSON values from an io.Reader as a stream.
type StreamReader struct {
	decoder *json.Decoder
	reader  *jsonReader
	r       *errorTrackingReader
}

// Next decodes and returns the next top-level JSON value.
//
// It returns io.EOF when the stream has reached a clean end with no more
// values. Whitespace-only input results in io.EOF without a value.
func (s *StreamReader) Next() (*model.Value, error) {
	if !s.decoder.More() {
		if s.r.err != nil {
			return nil, s.r.err
		}
		return nil, io.EOF
	}
	v, err := s.reader.decodeValue(s.decoder, 0)
	if err != nil {
		if s.r.err != nil {
			return nil, s.r.err
		}
		return nil, err
	}
	return v, nil
}

// errorTrackingReader remembers the last non-EOF read error because the
// underlying JSON decoder silently treats reader errors as end of input.
type errorTrackingReader struct {
	r   io.Reader
	err error
}

func (r *errorTrackingReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	if err != nil && err != io.EOF {
		r.err = err
	}
	return n, err
}

func (j *jsonReader) decodeValue(decoder *json.Decoder, depth int) (*model.Value, error) {
	t, err := decoder.Token()
	if err != nil {
		return nil, err
	}

	switch t {
	case jsonOpenObject:
		res, err := j.decodeObject(decoder, depth+1)
		if err != nil {
			return nil, fmt.Errorf("could not decode object: %w", err)
		}
		return res, nil
	case jsonOpenArray:
		res, err := j.decodeArray(decoder, depth+1)
		if err != nil {
			return nil, fmt.Errorf("could not decode array: %w", err)
		}
		return res, nil
	default:
		return j.decodeToken(decoder, t)
	}
}

func (j *jsonReader) decodeObject(decoder *json.Decoder, depth int) (*model.Value, error) {
	if depth > maxJSONDepth {
		return nil, ErrJSONMaxDepthExceeded
	}

	res := model.NewMapValue()

	var key any = nil

	for {
		t, err := decoder.Token()
		if err != nil {
			// We don't expect an EOF here since we're in the middle of processing an object.
			return res, err
		}

		switch t {
		case jsonOpenArray:
			if key == nil {
				return res, fmt.Errorf("unexpected token: %v", t)
			}
			value, err := j.decodeArray(decoder, depth+1)
			if err != nil {
				return res, err
			}
			if err := res.SetMapKey(key.(string), value); err != nil {
				return res, err
			}
			key = nil
		case jsonCloseArray:
			return res, fmt.Errorf("unexpected token: %v", t)
		case jsonCloseObject:
			return res, nil
		case jsonOpenObject:
			if key == nil {
				return res, fmt.Errorf("unexpected token: %v", t)
			}
			value, err := j.decodeObject(decoder, depth+1)
			if err != nil {
				return res, err
			}
			if err := res.SetMapKey(key.(string), value); err != nil {
				return res, err
			}
			key = nil
		default:
			if key == nil {
				if tStr, ok := t.(string); ok {
					key = tStr
				} else {
					return nil, fmt.Errorf("unexpected token: %v", t)
				}
			} else {
				value, err := j.decodeToken(decoder, t)
				if err != nil {
					return nil, err
				}
				if err := res.SetMapKey(key.(string), value); err != nil {
					return res, err
				}
				key = nil
			}
		}
	}
}

func (j *jsonReader) decodeArray(decoder *json.Decoder, depth int) (*model.Value, error) {
	if depth > maxJSONDepth {
		return nil, ErrJSONMaxDepthExceeded
	}

	res := model.NewSliceValue()
	for {
		t, err := decoder.Token()
		if err != nil {
			// We don't expect an EOF here since we're in the middle of processing an object.
			return res, err
		}

		switch t {
		case jsonOpenArray:
			value, err := j.decodeArray(decoder, depth+1)
			if err != nil {
				return res, err
			}
			if err := res.Append(value); err != nil {
				return res, err
			}
		case jsonCloseArray:
			return res, nil
		case jsonCloseObject:
			return res, fmt.Errorf("unexpected token: %t", t)
		case jsonOpenObject:
			value, err := j.decodeObject(decoder, depth+1)
			if err != nil {
				return res, err
			}
			if err := res.Append(value); err != nil {
				return res, err
			}
		default:
			value, err := j.decodeToken(decoder, t)
			if err != nil {
				return nil, err
			}
			if err := res.Append(value); err != nil {
				return res, err
			}
		}
	}
}

func (j *jsonReader) decodeToken(decoder *json.Decoder, t json.Token) (*model.Value, error) {
	switch tv := t.(type) {
	case json.Number:
		strNum := tv.String()
		// Numbers with a fraction or an exponent are floats.
		// E.g. 1.5, 1e3, 1E-2.
		if strings.ContainsAny(strNum, ".eE") {
			floatNum, err := tv.Float64()
			if err == nil {
				return model.NewFloatValue(floatNum), nil
			}
			return nil, err
		}
		intNum, err := tv.Int64()
		if err == nil {
			return model.NewIntValue(intNum), nil
		}

		return nil, err
	default:
		return model.NewValue(tv), nil
	}
}
