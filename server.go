package do

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path"
	"reflect"
	"regexp"
)

type serverConf struct {
	staticDir string
	addr      string
	port      string
}

type route struct {
	r       string
	cr      *regexp.Regexp
	method  string
	handler reflect.Value
}

type server struct {
	conf   *serverConf
	routes []route
	logger *log.Logger
}

func NewServer() *server {
	return &server{logger: log.New(os.Stdout, "[do]:", log.LstdFlags)}
}

func (s *server) addRout(r string, method string, handler interface{}) {
	cr, err := regexp.Compile(r)
	if err != nil {
		s.logger.Printf("error:route regex %q\n", r)
		return
	}
	switch handler.(type) {
	case reflect.Value:
		s.routes = append(s.routes, route{r: r, cr: cr, method: method, handler: handler.(reflect.Value)})
	default:
		s.routes = append(s.routes, route{r: r, cr: cr, method: method, handler: reflect.ValueOf(handler)})
	}
}

func (s *server) Get(route string, handler interface{}) {
	s.addRout(route, "GET", handler)
}

func (s *server) Post(route string, handler interface{}) {
	s.addRout(route, "POST", handler)
}

func (s *server) initServer() {
	if s.conf == nil {
		s.conf = &serverConf{}
	}
}

func (s *server) Run(addr string) {
	s.initServer()
	if err := http.ListenAndServe(addr, mainServer); err != nil {
		s.logger.Panicln("error in run:", err)
	}
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.process(w, r)
}

func (s *server) process(w http.ResponseWriter, r *http.Request) {
	s.routeHandler(w, r)
}

func fileExists(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func (s *server) tryServingFile(name string, w http.ResponseWriter, r *http.Request) bool {
	if s.conf.staticDir != "" {
		staticFile := path.Join(s.conf.staticDir, name)
		if fileExists(staticFile) {
			http.ServeFile(w, r, staticFile)
			return true
		}
	}
	return false
}

func (s *server) safelyCall() {

}

func requiresContext(handlerType reflect.Type) bool {
	if handlerType.NumIn() == 0 {
		return false
	}
	args0 := handlerType.In(0)
	if args0.Kind() != reflect.Ptr {
		return false
	}
	if args0.Elem() == reflect.TypeOf(Context{}) {
		return true
	}
	return false
}

func (s *server) routeHandler(w http.ResponseWriter, r *http.Request) {
	requestPath := r.URL.Path
	if r.Method == "GET" || r.Method == "HEAD" {
		if s.tryServingFile(requestPath, w, r) {
			return
		}
	}
	ctx := Context{ResponseWriter: w, Request: r, Params: map[string]string{}, server: s}
	r.ParseForm()
	if len(r.Form) > 0 {
		for k, v := range r.Form {
			ctx.Params[k] = v[0]
		}
	}
	for _, route := range s.routes {
		if !route.cr.MatchString(requestPath) {
			continue
		}
		match := route.cr.FindStringSubmatch(requestPath)
		if len(match[0]) != len(requestPath) {
			continue
		}
		var args []reflect.Value
		if requiresContext(route.handler.Type()) {
			args = append(args, reflect.ValueOf(&ctx))
		}
		for _, arg := range match[1:] {
			args = append(args, reflect.ValueOf(arg))
		}
		value := route.handler.Call(args)[0]
		if value.Kind() == reflect.String {
			if _, err := w.Write([]byte(value.String())); err != nil {
				s.logger.Println("error in write:", err)
			}
			return
		}
		data, err := json.Marshal(value.Interface())
		if err != nil {
			s.logger.Println("error in marshal data:", err)
		}
		if _, err := w.Write(data); err != nil {
			s.logger.Println("error in write:", err)
		}
	}
}
