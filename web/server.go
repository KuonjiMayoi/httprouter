package web

import (
	"net/http"
	"reflect"
	"regexp"
)

type Server struct {
	routes []route
}

type route struct {
	r string
	cr *regexp.Regexp
	method string
	handle reflect.Value
	httpHandle http.Handler
}

type Context struct{
	Request *http.Request
	Server *Server
	http.ResponseWriter
}


func (s *Server) addRoute(r string,method string,handle interface{}){
	cr,err := regexp.Compile(r)
	if err != nil{
		return
	}
	switch handle.(type) {
	case http.Handler:
		s.routes = append(s.routes, route{r:r,cr : cr,method:method,httpHandle : handle.(http.Handler)})
	case reflect.Value:
		s.routes = append(s.routes, route{r,cr,method,handle.(reflect.Value),nil})
	default:
		s.routes = append(s.routes, route{r,cr,method,reflect.ValueOf(handle),nil})
	}
}

func (s *Server) Post(r string,handle interface{}){
	s.addRoute(r,"POST",handle)
}
func (s *Server) Get(r string,handle interface{}){
	s.addRoute(r,"GET",handle)
}

func (s *Server) Run(addr string){
	//mux := http.NewServeMux()
	//mux.Handle("/",s)
	http.ListenAndServe(addr,s)
}


func (ctx *Context) SetHeader(hdr string, val string, unique bool) {
	if unique {
		ctx.Header().Set(hdr, val)
	} else {
		ctx.Header().Add(hdr, val)
	}
}
func (s *Server)ServeHTTP(w http.ResponseWriter,r *http.Request){
    s.Process(w,r)
}

func (s *Server)Process(w http.ResponseWriter,r *http.Request){
    s.routeHandle(w,r)
}

func (s *Server)routeHandle(w http.ResponseWriter,r *http.Request){
	reqPath := r.URL.Path
	r.ParseForm()
	ctx := &Context{r,s,w}
	for _,route := range s.routes{
        if !(route.method == r.Method) && !(route.method == "GET" && r.Method == "HEAD"){
        	continue
		}
		if !route.cr.MatchString(reqPath){
			continue
		}

        match := route.cr.FindStringSubmatch(reqPath)
        if len(match[0]) != len(reqPath){
        	continue
		}
        if route.httpHandle != nil{
        	route.httpHandle.ServeHTTP(w,r)
        	return
		}
		ctx.SetHeader("Content-Type", "text/html; charset=utf-8", true)
        var args []reflect.Value
		tp := route.handle.Type()
		if requireCtx(tp){
			args = append(args,reflect.ValueOf(ctx))
		}
		for _,arg := range match[1:]{
			args = append(args, reflect.ValueOf(arg))
		}
		_,err := s.safeCall(route.handle,args)
		if err != nil{
			ctx.Abort(500,"Server Error")
		}
	}
}

func (ctx *Context) Abort(status int, body string) {
	ctx.SetHeader("Content-Type", "text/html; charset=utf-8", true)
	ctx.ResponseWriter.WriteHeader(status)
	ctx.ResponseWriter.Write([]byte(body))
}
func (s *Server) safeCall(function reflect.Value,args []reflect.Value)(rst []reflect.Value,e interface{}){
	defer func() {
		if err := recover();err != nil{
			e = err
			rst = nil
			return
		}
	}()
	return function.Call(args),nil
}
func requireCtx(tp reflect.Type) bool {
	if tp.NumIn() <= 0{
		return false
	}
	if tp.In(0) != contextType{
		return false
	}
	return true
}

func Get(r string,handle interface{}){
	mainServer.Get(r,handle)
}

func Post(r string,handle interface{}){
	mainServer.Post(r,handle)
}

func Run(addr string){
	mainServer.Run(addr)
}

var contextType = reflect.TypeOf(&Context{})
var mainServer = &Server{}