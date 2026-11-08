package main

import (
 "bytes"
 "context"
 "errors"
 "fmt"
 "io"
 "os"
 "reflect"
 "strings"
 "sync/atomic"
 "time"

 "github.com/tomwright/dasel/v3/internal/cli"
 "github.com/tomwright/dasel/v3/execution"
 "github.com/tomwright/dasel/v3/model"
 jsonReader "github.com/tomwright/dasel/v3/parsing/json"
 _ "github.com/tomwright/dasel/v3/parsing/yaml"
)

var checks, unmet int
func check(name string, ok bool, detail string) {
 checks++;if !ok {unmet++};fmt.Printf("CHECK %s met=%v %s\n",name,ok,detail)
}
func run(args []string,r io.Reader,w io.Writer) error {
 old:=os.Args;defer func(){os.Args=old}();os.Args=append([]string{"dasel","query"},args...)
 _,err:=cli.Run(r,w,io.Discard);return err
}
func sample(name,input,want string,args ...string) {
 var out bytes.Buffer;err:=run(args,strings.NewReader(input),&out)
 check(name,err==nil&&out.String()==want,fmt.Sprintf("out=%q err=%v",out.String(),err))
}
type forbiddenReader struct{reads int}
func(r *forbiddenReader)Read(p []byte)(int,error){r.reads++;return 0,errors.New("unexpected stdin read")}
type notifyWriter struct{ch chan string}
func(w *notifyWriter)Write(p []byte)(int,error){w.ch<-string(p);return len(p),nil}
type gateWriter struct{entered chan struct{};release chan struct{};calls int;accepted bytes.Buffer;fail bool}
var sinkError=errors.New("controlled sink failure")
func(w *gateWriter)Write(p []byte)(int,error){w.calls++;if w.calls==1{w.accepted.Write(p);return len(p),nil};if w.calls==2{close(w.entered);<-w.release};if w.fail{w.accepted.Write(p[:1]);return 1,sinkError};w.accepted.Write(p);return len(p),nil}
func main(){
 for _,args:=range [][]string{{"--stream","-i","yaml"},{"--stream","-o","yaml"},{"--stream","--it"}}{r:=&forbiddenReader{};err:=run(args,r,io.Discard);check("R1_prevalidate",err!=nil&&r.reads==0,fmt.Sprintf("args=%q reads=%d err=%v",args,r.reads,err))}
 sample("R1_empty_no_query","","","--stream","missing()")
 sample("R1_whitespace"," \n\t","","--stream")
 pr,pw:=io.Pipe();w:=&notifyWriter{ch:make(chan string,2)};done:=make(chan error,1)
 go func(){done<-run([]string{"--stream","name"},pr,w)}()
 _,err:=pw.Write([]byte(`{"name":"open"}`));if err!=nil{panic(err)}
 select{case first:=<-w.ch:check("R1_open_pipe",first=="\"open\"\n",fmt.Sprintf("out=%q inputStillOpen=true",first));case<-time.After(3*time.Second):check("R1_open_pipe",false,"no output while input open")}
 pw.Close();err=<-done;pr.Close();check("R1_pipe_completion",err==nil,fmt.Sprintf("err=%v",err))
 sample("R2_mixed_values","{\n\"a\":1\n}\n[1,2]\n1 2 true false \"x\" null\n","{\"a\":1}\n[1,2]\n1\n2\ntrue\nfalse\n\"x\"\nnull\n","--stream","--compact")
 sample("R3_nested_var","{}{}{}","[2,3]\n[2,3]\n[2,3]\n","--stream","--compact","--var",`v=json:{"obj":{"n":1},"arr":[2]}`,`$v.obj.n = $v.obj.n + 1; $v.arr[0] = $v.arr[0] + 1; [$v.obj.n,$v.arr[0]]`)
 sample("R3_root_independent",`{"a":1}{"a":5}`,"{\"a\":2}\n{\"a\":6}\n","--stream","--compact","--root",`a = a + 1`)
 sample("R3_html_flag",`"<a>" "&"`,"\"\\u003ca\\u003e\"\n\"\\u0026\"\n","--stream","--write-flag","json-escape=html")
 sample("R3_rw_flag",`"<a>"`,"\"\\u003ca\\u003e\"\n","--stream","--rw-flag","json-escape=html")
 sample("R3_spread","[1,2][3]","1\n\n2\n3\n","--stream","...")
 sample("R3_branch","[1,2][3]","1\n\n2\n3\n","--stream","--unstable","branch(...)")
 var count atomic.Int64
 execution.DefaultFuncCollection["reviewTick"]=func(_ context.Context,data *model.Value,_ model.Values)(*model.Value,error){count.Add(1);return data,nil}
 defer delete(execution.DefaultFuncCollection,"reviewTick")
 var out bytes.Buffer
 err=run([]string{"--stream",`has("create") ? ($z = 9) : $z`},strings.NewReader(`{"create":true}{}`),&out)
 check("R3_new_var_absent",err!=nil&&strings.Contains(err.Error(),"record 2")&&strings.Contains(err.Error(),"variable z not found")&&out.String()=="9\n",fmt.Sprintf("out=%q err=%v",out.String(),err))
 for _,bad:=range []string{`{"a":`, `[1,`, `]`, `}`, `{bad}`} {
  out.Reset();count.Store(0);err=run([]string{"--stream","reviewTick(); a"},strings.NewReader(`{"a":1}`+bad),&out)
  check("R4_parse_"+bad,err!=nil&&strings.Contains(err.Error(),"record 2")&&out.String()=="1\n"&&count.Load()==1,fmt.Sprintf("bad=%q out=%q queries=%d err=%v",bad,out.String(),count.Load(),err))
 }
 count.Store(0);out.Reset();err=run([]string{"--stream",`reviewTick(); toInt(a)`},strings.NewReader(`{"a":"1"}{"a":"bad"}{"a":"3"}`),&out)
 check("R4_query_failure",err!=nil&&strings.Contains(err.Error(),"record 2")&&strings.Contains(err.Error(),"invalid syntax")&&out.String()=="1\n"&&count.Load()==2,fmt.Sprintf("out=%q queries=%d err=%v",out.String(),count.Load(),err))
 sample("R4_fresh_invocation",`{"a":"2"}`,"3\n","--stream","--var","v=json:1",`$v = $v + toInt(a)`)
 out.Reset();count.Store(0);err=run([]string{"--stream","--compact",`reviewTick(); [a, 1.0 / a]`},strings.NewReader(`{"a":1}{"a":0}{"a":2}`),&out)
 check("R5_real_encoding_failure",err!=nil&&strings.Contains(err.Error(),"record 2")&&strings.Contains(err.Error(),"+Inf")&&out.String()=="[1,1]\n"&&count.Load()==2,fmt.Sprintf("out=%q queries=%d err=%v",out.String(),count.Load(),err))
 for _,fail:=range []bool{false,true}{
  count.Store(0);sink:=&gateWriter{entered:make(chan struct{}),release:make(chan struct{}),fail:fail};finished:=make(chan error,1)
  go func(){finished<-run([]string{"--stream",`reviewTick(); a`},strings.NewReader(`{"a":1}{"a":2}{"a":3}`),sink)}()
  select{case<-sink.entered:case<-time.After(3*time.Second):panic("writer never reached")}
  atBlock:=count.Load();close(sink.release);err=<-finished
  ok:=atBlock==2;if fail{ok=ok&&errors.Is(err,sinkError)&&count.Load()==2&&sink.calls==2&&sink.accepted.String()=="1\n2"}else{ok=ok&&err==nil&&count.Load()==3&&sink.accepted.String()=="1\n2\n3\n"}
  check(fmt.Sprintf("R6_backpressure_fail_%v",fail),ok,fmt.Sprintf("queriesAtBarrier=%d finalQueries=%d writes=%d accepted=%q err=%v",atBlock,count.Load(),sink.calls,sink.accepted.String(),err))
 }
 raw:=bytes.Repeat([]byte(`{"x":"012345678901234567890123456789"}`+"\n"),32768)
 sr:=jsonReader.NewStreamReader(bytes.NewReader(raw));var small,large,cursor int64
 for i:=1;i<=32768;i++ {if _,e:=sr.Next();e!=nil{panic(e)};if i==1024||i==32768 {s:=reflect.ValueOf(sr).Elem().FieldByName("decoder").Elem().FieldByName("s").Elem();n:=int64(s.FieldByName("buf").Len());cursor=s.FieldByName("cursor").Int();if i==1024{small=n}else{large=n}}}
 check("R6_bounded_input_storage",large<small*4,fmt.Sprintf("records=1024 buffer=%d records=32768 buffer=%d retainedCursor=%d totalInputBytes=%d",small,large,cursor,len(raw)))
 sample("R7_batch","{\"a\":1}{\"a\":2}","1\n\n2\n","-i","json","a")
 sample("R7_empty","","null\n","-i","json")
 sample("R7_json_to_yaml",`{"a":1}`,"a: 1\n","-i","json","-o","yaml")
 fmt.Printf("INDEPENDENT SUMMARY checks=%d unmet=%d\n",checks,unmet)
}
