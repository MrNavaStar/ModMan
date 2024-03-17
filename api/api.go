package api

import (
	"github.com/go-resty/resty/v2"
)

var client = resty.New()

type Version struct {
	Version string
	Stable  bool
	Url     string
}
