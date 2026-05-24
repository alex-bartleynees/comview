module github.com/rockorager/comview

go 1.26.1

require (
	github.com/alecthomas/chroma/v2 v2.24.1
	github.com/rockorager/go-uucode v1.2.0
	go.rockorager.dev/vaxis v0.15.1-0.20260516233705-b848ef0bd8ce
)

replace go.rockorager.dev/vaxis => ../vaxis

require (
	github.com/dlclark/regexp2 v1.12.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
	golang.org/x/term v0.10.0 // indirect
)
