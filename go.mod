module example.com/gowebgo

go 1.17

require (
	github.com/go-httpproxy/httpproxy v0.0.0-20180417134941-6977c68bf38e
	golang.org/x/term v0.2.0
)

require golang.org/x/sys v0.2.0 // indirect

replace github.com/go-httpproxy/httpproxy v0.0.0-20180417134941-6977c68bf38e => ../httpproxy
