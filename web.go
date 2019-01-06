package do

func Get(route string, handler interface{}) {
	mainServer.Get(route, handler)
}

func Post(route string, handler interface{}) {
	mainServer.Post(route, handler)
}

func Run(addr string) {
	mainServer.Run(addr)
}

var mainServer = NewServer()
