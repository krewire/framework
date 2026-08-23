module github.com/krewire/framework

go 1.22

require github.com/krewire/libs v0.1.0

require (
	github.com/gomarkdown/markdown v0.0.0-20260818103853-6d1f24fc3a11
	golang.org/x/net v0.30.0
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/krewire/libs => ../libs
