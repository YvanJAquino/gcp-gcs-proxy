package auth

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"golang.org/x/oauth2"
	// "golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
)

const ScopePlatform string = "https://www.googleapis.com/auth/cloud-platform"

type AuthProxy struct {
	source oauth2.TokenSource
	token  *oauth2.Token
	proxy  *httputil.ReverseProxy
}

// Constructors

func New(ctx context.Context, target string) *AuthProxy {
	log.Printf("TargetURL: %s", target)
	targetURL, err := url.Parse(target)
	if err != nil {
		panic(err)
	}

	source, err := idtoken.NewTokenSource(ctx, targetURL.String())
	if err != nil {
		panic(err)
	}

	token, err := source.Token()
	if err != nil {
		panic(err)
	}

	proxy := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			r.Host = targetURL.Host
			r.URL.Scheme = targetURL.Scheme
			r.URL.Host = targetURL.Host
			r.URL.Path = targetURL.Path + r.URL.Path
			token.SetAuthHeader(r)
			go func() {

				b, err := httputil.DumpRequestOut(r, false)
				if err != nil {
					log.Println(err)
				} else {
					log.Println(string(b))
				}
				log.Println(token.Expiry)
				log.Println(r.URL.String())

			}()

		},
	}
	return &AuthProxy{
		source: source,
		token:  token,
		proxy:  proxy,
	}
}

// Public Methods

func (p *AuthProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.proxy.ServeHTTP(w, r)
}
