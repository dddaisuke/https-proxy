package main

import (
	"crypto/tls"
	"fmt"
	"github.com/dddaisuke/httputil2"
	"github.com/gorilla/mux"
	"github.com/srtkkou/zgok"
	"github.com/yhat/wsutil"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type Certificates struct {
	CertFile string
	KeyFile  string
}

func loadX509KeyPair(certFile, keyFile string) (tls.Certificate, error) {
	certPEMBlock, err := readFromZfs(certFile)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPEMBlock, err := readFromZfs(keyFile)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.X509KeyPair(certPEMBlock, keyPEMBlock)
}

func readFromZfs(filePath string) ([]byte, error) {
	zfs, _ := zgok.RestoreFileSystem(os.Args[0])
	if zfs != nil {
		// 本番環境
		return zfs.ReadFile(filePath)
	} else {
		// 開発環境
		return ioutil.ReadFile(filePath)
	}
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
		config.Certificates[i], err = loadX509KeyPair(v.CertFile, v.KeyFile)
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
	remote3000, err3000 := url.Parse("http://127.0.0.1:3000")
	if err3000 != nil {
		panic(err3000)
	}

	return httputil.NewSingleHostReverseProxy(remote3000)
}

func proxy3001() *httputil2.ReverseProxy {
	remote3001, err3001 := url.Parse("http://127.0.0.1:3001")
	if err3001 != nil {
		panic(err3001)
	}

	return httputil2.NewSingleHostReverseProxy(remote3001)
}

func proxy8080() *httputil.ReverseProxy {
	remote8080, err8080 := url.Parse("http://127.0.0.1:8080")
	if err8080 != nil {
		panic(err8080)
	}

	return httputil.NewSingleHostReverseProxy(remote8080)
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

	backendURL := &url.URL{Scheme: "ws://", Host: "127.0.0.1:8080"}
	p := wsutil.NewSingleHostReverseProxy(backendURL)
	route.Host("board.manabo.info").PathPrefix("/socket.io/1/websocket/").Handler(p)
	route.Host("board.manabo.info").PathPrefix("/socket.io/").Handler(proxy8080())
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
