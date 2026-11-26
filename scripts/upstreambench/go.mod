module github.com/nao1215/imaging/scripts/upstreambench

go 1.26.0

replace github.com/nao1215/imaging => ../..

require (
	github.com/disintegration/imaging v1.6.3-0.20201218193011-d40f48ce0f09
	github.com/nao1215/imaging v1.0.13
)

require (
	golang.org/x/image v0.48.0 // indirect
	golang.org/x/sync v0.25.0 // indirect
)
