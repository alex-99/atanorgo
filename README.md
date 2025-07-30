# atanorgo

Первая версия проекта Atanor

## Установка

```bash
go get github.com/alexandr-lavrev/atanorgo
```

## Использование

```go
package main

import (
	"github.com/alexandr-lavrev/atanorgo/atanorgo"
	"github.com/alexandr-lavrev/atanorgo/atanorgo/pkg/utils"
) 

func main() {
	atanorgo.SayHello("Alex")
	utils.SayHello("Alex")
}
```

## Разработка