package goproxy

import (
	"crypto/tls"
	"net/http"
	"regexp"
)

// ProxyCtx is the Proxy context, contains useful information about every request. It is passed to
// every user function. Also used as a logger.
type ProxyCtx struct {
	Req          *http.Request  // Client request to the proxy
	Resp         *http.Response // Remote server's response (nil if the request wasn't send yet)
	Websocket    bool           // true if Connection is a Websocket
	RoundTripper RoundTripper
	Error        error       // The recent error that occurred while trying to send receive or parse traffic
	UserData     interface{} // User data kept in the context, from the call of ReqHandler to the call of RespHandler
	Session      int64       // Invariant from a request to a response
	signer       func(ca *tls.Certificate, hostname []string) (*tls.Certificate, error)
	proxy        *ProxyHttpServer
}

type RoundTripper interface {
	RoundTrip(req *http.Request, ctx *ProxyCtx) (*http.Response, error)
}

type RoundTripperFunc func(req *http.Request, ctx *ProxyCtx) (*http.Response, error)

func (f RoundTripperFunc) RoundTrip(req *http.Request, ctx *ProxyCtx) (*http.Response, error) {
	return f(req, ctx)
}

func (ctx *ProxyCtx) RoundTrip(req *http.Request) (*http.Response, error) {
	if ctx.RoundTripper != nil {
		return ctx.RoundTripper.RoundTrip(req, ctx)
	}
	return ctx.proxy.Tr.RoundTrip(req)
}

func (ctx *ProxyCtx) printf(msg string, argv ...interface{}) {
	ctx.proxy.Logger.Printf("[%03d] "+msg+"\n", append([]interface{}{ctx.Session & 0xFF}, argv...)...)
}

// Logf prints a message to the proxy's log. Should be used in a ProxyHttpServer's filter
// This message will be printed only if the Verbose field of the ProxyHttpServer is set to true
//
//	proxy.OnRequest().DoFunc(func(r *http.Request,ctx *goproxy.ProxyCtx) (*http.Request, *http.Response){
//		nr := atomic.AddInt32(&counter,1)
//		ctx.Printf("So far %d requests",nr)
//		return r, nil
//	})
func (ctx *ProxyCtx) Logf(msg string, argv ...interface{}) {
	if ctx != nil && ctx.proxy != nil && ctx.proxy.Verbose {
		ctx.printf("INFO: "+msg, argv...)
	}
}

// Warnf prints a message to the proxy's log. Should be used in a ProxyHttpServer's filter
// This message will always be printed.
//
//	proxy.OnRequest().DoFunc(func(r *http.Request,ctx *goproxy.ProxyCtx) (*http.Request, *http.Response){
//		f,err := os.OpenFile(cachedContent)
//		if err != nil {
//			ctx.Warnf("error open file %v: %v",cachedContent,err)
//			return r, nil
//		}
//		return r, nil
//	})
func (ctx *ProxyCtx) Warnf(msg string, argv ...interface{}) {
	if ctx != nil && ctx.proxy != nil {
		ctx.printf("WARN: "+msg, argv...)
	}
}

var charsetFinder = regexp.MustCompile("charset=([^ ;]*)")

// Will try to infer the character set of the request from the headers.
// Returns the empty string if we don't know which character set it used.
// Currently it will look for charset=<charset> in the Content-Type header of the request.
func (ctx *ProxyCtx) Charset() string {
	charsets := charsetFinder.FindStringSubmatch(ctx.Resp.Header.Get("Content-Type"))
	if charsets == nil {
		return ""
	}
	return charsets[1]
}
