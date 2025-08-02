package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/alex-99/atanorgo/pkg/avptcp"
)

func main() {
	fmt.Println("Hello, TCP")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Создаем контекст с отменой
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запускаем клиентов
	for i := 0; i < 90; i++ {
		tcp := avptcp.NewTcpClient("127.0.0.1", "5555")
		tcp.Name = strconv.Itoa(i)
		tcp.Debug = true
		tcp.Delim = 9
		tcp.Telnet = true
		tcp.HelloMessage = "Hi\n"
		tcp.OnConnect = func() {
			fmt.Println("Connected")
			tcp.Send([]byte("Hello, TCP " + tcp.Name + "\n"))
			// time.Sleep(500 * time.Millisecond)
			// tcp.Stop()
			// cancel()
		}
		tcp.OnClose = func() {
			fmt.Println("Closed")
		}
		tcp.OnMessage = func(msg []byte) {
			fmt.Println("Received "+tcp.Name+":", msg)
		}
		// go StartTcp(ctx, "127.0.0.1", "5555")
		tcp.StartReconnectGo(ctx)

		// fmt.Println("Closed")

	}
	// Ожидаем либо сигнал, либо таймаут
	select {
	case <-sigChan:
		fmt.Println("\nReceived shutdown signal")
	case <-time.After(130 * time.Second):
		fmt.Println("Timeout reached")
	}

	// Инициируем остановку всех горутин
	cancel()

	time.Sleep(500 * time.Millisecond)
	fmt.Println("Bye")
}
