package main

import (
 "context"
 "fmt"
 "reflect"
 "strings"
 dasel "github.com/tomwright/dasel/v3"
 "github.com/tomwright/dasel/v3/execution"
 "github.com/tomwright/dasel/v3/model"
 "github.com/tomwright/dasel/v3/model/orderedmap"
)

func main() {
 ctx:=context.Background()
 checks:=0
 check:=func(label string, ok bool, actual any) { checks++; fmt.Printf("%s ok=%v actual=%v\n",label,ok,actual); if !ok { panic(label) } }
 run:=func(q string,in *model.Value,o *execution.Options) *model.Value {v,e:=execution.ExecuteSelector(ctx,q,in,o);if e!=nil {panic(e)};return v}
 eq:=func(v *model.Value,w any)bool {b,e:=v.EqualTypeValue(model.NewValue(w));return b&&e==nil}
 null:=model.NewNullValue()
 api,_,apiErr:=dasel.Select(ctx,orderedmap.NewMap().Set("n",40),`def bump($x) { $x + 1 }; def two($x) { bump(bump($x)) }; two(n)`)
 check("R1 public Go API nonrecursive chain",apiErr==nil,fmt.Sprintf("result=%v err=%v",api,apiErr))
 o:=execution.NewOptions(execution.WithVariable("x",model.NewIntValue(7)))
 v:=run(`def inner($p) { $x = 99; $p = 88; $local = 3; $p }; def outer($p) { $x = 11; inner($p); [$x, $p] }; [outer(5), $x]`,null,o)
 check("R2 nested parameter and ordinary binding",eq(v,[]any{[]any{11,5},7}),v)
 _,local:=o.Vars["local"];check("R2 host unchanged",!local&&eq(o.Vars["x"],7),o.Vars["x"])
 root:=model.NewValue(orderedmap.NewMap().Set("value",1))
 o=execution.NewOptions(execution.WithVariable("root",root),execution.WithVariable("x",model.NewIntValue(7)))
 v=run(`def mutate($obj) { $obj.value = 2; $obj = {value: 90}; $obj.value = 91 }; mutate($root); $root.value`,root,o)
 check("R3 field write survives object rebinding",eq(v,2),v)
 v=run(`def idx() { $key }; def outer() { $saved = $key; $root.arr.map(idx()); [$saved, $key] }; $root.arr.map(outer())`,model.NewValue([]any{}),execution.NewOptions(execution.WithVariable("root",model.NewValue(orderedmap.NewMap().Set("arr",[]any{10,20})))))
 check("R3 nested key restored",eq(v,[]any{[]any{0,0},[]any{1,1}}),v)
 v=run(`def node() { $this }; def eachNode() { map(node()) }; map(eachNode())`,model.NewValue([]any{[]any{1,2},[]any{3}}),execution.NewOptions())
 check("R3 nested this",eq(v,[]any{[]any{1,2},[]any{3}}),v)
 _,err:=execution.ExecuteSelector(ctx,`def inner($p) { $x = 99; $local = 3; $root.value = 4; toInt("bad") }; def outer() { $x = 11; inner(5) }; outer()`,root,o)
 check("R5 nested actual conversion error",err!=nil&&strings.Contains(err.Error(),"bad"),err)
 check("R5 host binding and finished field write",eq(o.Vars["x"],7)&&eq(run(`$root.value`,root,o),4),root)
 _,local=o.Vars["local"];check("R5 no local",!local,local)
 _,err=execution.ExecuteSelector(ctx,`outer()`,root,o);check("R5 no stale definition",err!=nil&&strings.Contains(err.Error(),"unknown function"),err)
 v=run(`def inner($p) { toInt($p) }; def outer() { inner("42") }; outer()`,root,o);check("R5 corrected same Options",eq(v,42),v)
 native:=execution.DefaultFuncCollection.Copy();count:=0
 native["tick"]=func(_ context.Context,d *model.Value,_ model.Values)(*model.Value,error){count++;return d.GetMapKey("n")}
 original:=native.Copy();ptrs:=func(fc execution.FuncCollection)map[string]uintptr{m:=map[string]uintptr{};for k,f:=range fc {m[k]=reflect.ValueOf(f).Pointer()};return m}
 o=execution.NewOptions(execution.WithFuncs(native))
 v=run(`def twice($a) { $a + $a }; def wrapped($a) { toInt($a) }; map(twice(wrapped(tick())))`,model.NewValue([]any{orderedmap.NewMap().Set("n","2"),orderedmap.NewMap().Set("n","3")}),o)
 check("R7 one evaluation on caller nodes",count==2&&eq(v,[]any{4,6}),fmt.Sprintf("ticks=%d value=%v",count,v))
 check("R7 native collection unchanged on success",reflect.DeepEqual(ptrs(native),ptrs(original)),len(native))
 _,err=execution.ExecuteSelector(ctx,`def bad() { toInt("bad") }; bad()`,null,o)
 check("R7 native collection unchanged on failure",err!=nil&&reflect.DeepEqual(ptrs(native),ptrs(original)),len(native))
 _,err=execution.ExecuteSelector(ctx,`def tick() { 1 }; tick()`,null,o);check("R4 custom native conflict",err!=nil&&strings.Contains(err.Error(),"native function"),err)
 _,err=execution.ExecuteSelector(ctx,`$x = 9; toInt("bad")`,null,execution.NewOptions(execution.WithVariable("x",model.NewIntValue(7))))
 o=execution.NewOptions(execution.WithVariable("x",model.NewIntValue(7)))
 _,err=execution.ExecuteSelector(ctx,`$x = 9; toInt("bad")`,null,o)
 check("R6 global assignment persists after error",err!=nil&&eq(o.Vars["x"],9)&&eq(run(`$x+1`,null,o),10),o.Vars["x"])
 fmt.Printf("INDEPENDENT CHECKS %d PASS\n",checks)
}
