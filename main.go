package main

import (
	"crypto/tls"
	"fmt"
	"github.com/gorilla/mux"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Certificates struct {
	CertFile string
	KeyFile  string
}

func listenAndServeTLSSNI(srv *http.Server, certs []Certificates) error {
	addr := srv.Addr
	if addr == "" {
		addr = ":https"
	}
	config := &tls.Config{}
	if srv.TLSConfig != nil {
		*config = *srv.TLSConfig
	}
	if config.NextProtos == nil {
		config.NextProtos = []string{"http/1.1"}
	}

	var err error

	config.Certificates = make([]tls.Certificate, len(certs))
	for i, v := range certs {
		config.Certificates[i], err = tls.LoadX509KeyPair(v.CertFile, v.KeyFile)
		if err != nil {
			return err
		}
	}

	config.BuildNameToCertificate()

	conn, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	tlsListener := tls.NewListener(conn, config)
	return srv.Serve(tlsListener)
}

func proxy3000() *httputil.ReverseProxy {
	remote3000, err3000 := url.Parse("http://lvh.me:3000")
	if err3000 != nil {
		panic(err3000)
	}

	return httputil.NewSingleHostReverseProxy(remote3000)
}

func proxy3001() *httputil.ReverseProxy {
	remote3001, err3001 := url.Parse("http://lvh.me:3001")
	if err3001 != nil {
		panic(err3001)
	}

	return httputil.NewSingleHostReverseProxy(remote3001)
}

func certManaboInfo() Certificates {
	return Certificates{
		CertFile: "ssl/manabo-info/fullchain.pem",
		KeyFile:  "ssl/manabo-info/privkey.pem",
	}
}

func certBoardManaboInfo() Certificates {
	return Certificates{
		CertFile: "ssl/board-manabo-info/fullchain.pem",
		KeyFile:  "ssl/board-manabo-info/privkey.pem",
	}
}

func main() {
	var certs []Certificates

	certs = append(certs, certManaboInfo())
	certs = append(certs, certBoardManaboInfo())

	route := mux.NewRouter()
	route.Host("manabo.info").PathPrefix("/").Handler(proxy3000())
	route.Host("board.manabo.info").PathPrefix("/").Handler(proxy3001())

	httpsServer := &http.Server{
		Addr:    ":443",
		Handler: route,
	}

	fmt.Println("Listening on port 443...")
	err := listenAndServeTLSSNI(httpsServer, certs)
	if err != nil {
		fmt.Printf("err: %s", err)
	}
}
