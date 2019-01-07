package do

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path"
	"reflect"
	"regexp"
	"time"
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
	return &server{logger: log.New(os.Stdout, "[do]:", log.LstdFlags|log.Lshortfile)}
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
	// 请求路径
	requestPath := r.URL.Path
	// 构建上下文
	ctx := Context{ResponseWriter: w, Request: r, querys: map[string]string{}, server: s}
	// 设置请求时间头
	ctx.SetHeader("Date", time.Now().Format(time.RFC1123), true)
	// 静态文件
	if r.Method == "GET" || r.Method == "HEAD" {
		if s.tryServingFile(requestPath, w, r) {
			return
		}
	}
	// 路由适配
	for _, route := range s.routes {
		if !route.cr.MatchString(requestPath) || route.method != r.Method {
			continue
		}
		// 路由匹配成功,获取路径参数
		match := route.cr.FindStringSubmatch(requestPath)
		if len(match[0]) != len(requestPath) {
			continue
		}
		var args []reflect.Value
		// 回调函数确认是否添加上下文参数
		if requiresContext(route.handler.Type()) {
			args = append(args, reflect.ValueOf(&ctx))
		}
		// 回调函数添加路径参数(有路径参数,回调函数必须接收,否则会产生异常)
		for _, arg := range match[1:] {
			args = append(args, reflect.ValueOf(arg))
		}
		// 调用处理函数,获取响应数据
		value := route.handler.Call(args)[0]
		// 响应html文本数据
		if value.Kind() == reflect.String { //Content-Type →text/html; charset=utf-8 =>自动设置
			if _, err := w.Write([]byte(value.String())); err != nil {
				s.logger.Println("error in write:", err)
			}
			return
		}
		// 响应json数据
		data, err := json.Marshal(value.Interface())
		if err != nil {
			s.logger.Println("error in marshal data:", err)
		}
		ctx.SetHeader("Content-Type", "application/json", true)
		if _, err := w.Write(data); err != nil {
			s.logger.Println("error in write:", err)
		}
	}
}
