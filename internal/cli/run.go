package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/tomwright/dasel/v3/execution"
	"github.com/tomwright/dasel/v3/model"
	"github.com/tomwright/dasel/v3/parsing"
	"github.com/tomwright/dasel/v3/parsing/json"
)

type runOpts struct {
	Vars              variables
	ExtReadWriteFlags extReadWriteFlags
	ExtReadFlags      extReadWriteFlags
	ExtWriteFlags     extReadWriteFlags
	InFormat          string
	OutFormat         string
	Compact           bool
	ReturnRoot        bool
	Unstable          bool
	Query             string

	ConfigPath string

	Stdin io.Reader
}

func run(o runOpts) ([]byte, error) {
	cfg, err := LoadConfig(o.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("error loading config: %w", err)
	}

	var opts []execution.ExecuteOptionFn

	if o.OutFormat == "" && o.InFormat != "" {
		o.OutFormat = o.InFormat
	} else if o.OutFormat != "" && o.InFormat == "" {
		o.InFormat = o.OutFormat
	} else if o.OutFormat == "" {
		o.OutFormat = cfg.DefaultFormat
	}

	readerOptions := parsing.DefaultReaderOptions()
	applyReaderFlags(&readerOptions, o.ExtReadFlags, o.ExtReadWriteFlags)

	var reader parsing.Reader
	if len(o.InFormat) > 0 {
		reader, err = parsing.Format(o.InFormat).NewReader(readerOptions)
		if err != nil {
			return nil, fmt.Errorf("failed to get input reader: %w", err)
		}
	}

	writerOptions := parsing.DefaultWriterOptions()
	writerOptions.Compact = o.Compact
	applyWriterFlags(&writerOptions, o.ExtWriteFlags, o.ExtReadWriteFlags)

	writer, err := parsing.Format(o.OutFormat).NewWriter(writerOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get output writer: %w", err)
	}

	opts = append(opts, variableOptions(o.Vars)...)

	// Default to null. If stdin is being read then this will be overwritten.
	inputData := model.NewNullValue()

	var inputBytes []byte
	if o.Stdin != nil {
		inputBytes, err = io.ReadAll(o.Stdin)
		if err != nil {
			return nil, fmt.Errorf("error reading stdin: %w", err)
		}
	}

	if len(inputBytes) > 0 {
		if reader == nil {
			return nil, fmt.Errorf("input format is required when reading stdin")
		}
		inputData, err = reader.Read(inputBytes)
		if err != nil {
			return nil, fmt.Errorf("error reading input: %w", err)
		}
	}

	opts = append(opts, execution.WithVariable("root", inputData))

	if o.Unstable {
		opts = append(opts, execution.WithUnstable())
	}

	options := execution.NewOptions(opts...)
	out, err := execution.ExecuteSelector(context.Background(), o.Query, inputData, options)
	if err != nil {
		return nil, err
	}

	if o.ReturnRoot {
		out = inputData
	}

	outputBytes, err := writer.Write(out)
	if err != nil {
		return nil, fmt.Errorf("error writing output: %w", err)
	}

	return outputBytes, nil
}

// runStream runs a query against a stream of independent top-level JSON
// values read from stdin.
//
// Each value is treated as one input record. A record is fully parsed,
// queried and serialised before the next record is processed, and the
// result is written to stdout before continuing, so a blocked output
// prevents further queries. Every record starts with its own $root and a
// fresh copy of the initial --var values.
func runStream(o runOpts, stdin io.Reader, stdout io.Writer) error {
	writerOptions := parsing.DefaultWriterOptions()
	writerOptions.Compact = o.Compact
	applyWriterFlags(&writerOptions, o.ExtWriteFlags, o.ExtReadWriteFlags)

	// Format validation happens before runStream is called.
	writer, err := json.JSON.NewWriter(writerOptions)
	if err != nil {
		return fmt.Errorf("failed to get output writer: %w", err)
	}

	if stdin == nil {
		// An empty stream executes no queries and outputs nothing.
		return nil
	}

	var initialVars []variable
	if o.Vars != nil {
		initialVars = *o.Vars
	}

	streamReader := json.NewStreamReader(stdin)

	recordNum := 0
	for {
		inputData, err := streamReader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		recordNum++
		if err != nil {
			return fmt.Errorf("error parsing input record %d: %w", recordNum, err)
		}

		opts := make([]execution.ExecuteOptionFn, 0, len(initialVars)+2)
		for _, v := range initialVars {
			varCopy, err := copyStreamValue(v.Value)
			if err != nil {
				return fmt.Errorf("error preparing variable %q for input record %d: %w", v.Name, recordNum, err)
			}
			opts = append(opts, execution.WithVariable(v.Name, varCopy))
		}
		opts = append(opts, execution.WithVariable("root", inputData))
		if o.Unstable {
			opts = append(opts, execution.WithUnstable())
		}

		out, err := execution.ExecuteSelector(context.Background(), o.Query, inputData, execution.NewOptions(opts...))
		if err != nil {
			return fmt.Errorf("error executing query on input record %d: %w", recordNum, err)
		}

		if o.ReturnRoot {
			out = inputData
		}

		// Serialise the whole result before publishing any of it, so a
		// failed record never produces a partial output.
		outputBytes, err := writer.Write(out)
		if err != nil {
			return fmt.Errorf("error serialising result for input record %d: %w", recordNum, err)
		}

		// Publish the complete result, in input order, before reading or
		// querying the next record. A blocked writer applies backpressure.
		if _, err := stdout.Write(outputBytes); err != nil {
			return fmt.Errorf("error writing output for input record %d: %w", recordNum, err)
		}
	}
}

// copyStreamValue returns a deep, independent copy of v so that mutations
// made while processing one stream record cannot leak into the next one.
func copyStreamValue(v *model.Value) (*model.Value, error) {
	switch v.Type() {
	case model.TypeMap:
		res := model.NewMapValue()
		if err := v.RangeMap(func(k string, cv *model.Value) error {
			cp, err := copyStreamValue(cv)
			if err != nil {
				return err
			}
			return res.SetMapKey(k, cp)
		}); err != nil {
			return nil, err
		}
		return res, nil
	case model.TypeSlice:
		res := model.NewSliceValue()
		if v.IsBranch() {
			res.MarkAsBranch()
		}
		if v.IsSpread() {
			res.MarkAsSpread()
		}
		if err := v.RangeSlice(func(_ int, cv *model.Value) error {
			cp, err := copyStreamValue(cv)
			if err != nil {
				return err
			}
			return res.Append(cp)
		}); err != nil {
			return nil, err
		}
		return res, nil
	case model.TypeString:
		val, err := v.StringValue()
		if err != nil {
			return nil, err
		}
		return model.NewStringValue(val), nil
	case model.TypeInt:
		val, err := v.IntValue()
		if err != nil {
			return nil, err
		}
		return model.NewIntValue(val), nil
	case model.TypeFloat:
		val, err := v.FloatValue()
		if err != nil {
			return nil, err
		}
		return model.NewFloatValue(val), nil
	case model.TypeBool:
		val, err := v.BoolValue()
		if err != nil {
			return nil, err
		}
		return model.NewBoolValue(val), nil
	case model.TypeNull:
		return model.NewNullValue(), nil
	default:
		return nil, fmt.Errorf("cannot copy value of type %s", v.Type())
	}
}
