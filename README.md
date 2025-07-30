# atanorgo

Первая версия проекта Atanor

git tag v1.0.2
git push origin v1.0.2
go get github.com/alex-99/atanorgo@latest  

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

для тестирования
cmd/tcp/main.go  
ncat -v -k  -l -p 5555