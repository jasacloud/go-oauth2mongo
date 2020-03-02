# Mongo Storage for [OAuth 2.0](https://github.com/go-oauth2/oauth2) with official driver

[![Build][Build-Status-Image]][Build-Status-Url] [![Codecov][codecov-image]][codecov-url] [![ReportCard][reportcard-image]][reportcard-url] [![GoDoc][godoc-image]][godoc-url] [![License][license-image]][license-url]

## Install

``` bash
$ go get -u -v gopkg.in/go-oauth2/mongo.v3
```

## Usage

``` go
package main

import (
	"github.com/jasacloud/go-libraries/db/mongoc"
	"github.com/jasacloud/go-oauth2mongo"
	"gopkg.in/oauth2.v3/manage"
)

func main() {
	manager := manage.NewDefaultManager()

	conn, err := mongoc.NewConnection("accounts")
	if err != nil {
		panic(err)
	}

	// use mongodb token store
	manager.MapTokenStorage(
		oauth2mongo.NewTokenStore(conn),
	)
	manager.MapClientStorage(
		oauth2mongo.NewClientStore(conn),
	)
	// ...
}
```

## MIT License

```
Copyright (c) 2020 Jasacloud
```

[Build-Status-Url]: https://travis-ci.org/jasacloud/go-oauth2mongo
[Build-Status-Image]: https://travis-ci.org/jasacloud/go-oauth2mongo.svg?branch=master
[codecov-url]: https://codecov.io/gh/jasacloud/go-oauth2mongo
[codecov-image]: https://codecov.io/gh/jasacloud/go-oauth2mongo/branch/master/graph/badge.svg
[reportcard-url]: https://goreportcard.com/report/github.com/jasacloud/go-oauth2mongo
[reportcard-image]: https://goreportcard.com/badge/github.com/jasacloud/go-oauth2mongo
[godoc-url]: https://godoc.org/github.com/jasacloud/go-oauth2mongo
[godoc-image]: https://godoc.org/github.com/jasacloud/go-oauth2mongo?status.svg
[license-url]: http://opensource.org/licenses/MIT
[license-image]: https://img.shields.io/npm/l/express.svg
