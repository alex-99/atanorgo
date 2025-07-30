package avptcp

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"time"
)

// TcpClient is a middleman between the websocket connection and the hub.
type TcpClient struct {
	ip             string
	port           string
	reconnect      bool
	Name           string
	Conn           net.Conn
	OnMessage      func(msg []byte)
	OnClose        func()
	OnConnect      func()
	reconnectDelay int
}

// NewTcpClient создает новый экземпляр TcpClient.
func NewTcpClient(ip string, port string) *TcpClient {
	tcpClient := &TcpClient{
		ip:             ip,
		port:           port,
		reconnectDelay: 3,
		//reconnect: true,
	}
	return tcpClient
}

func (tcpClient *TcpClient) StartReconnect(ctx context.Context) {
	fmt.Printf("TCP client %s  %s started\n", tcpClient.ip, tcpClient.port)

	defer func() {
		tcpClient.Stop()
		fmt.Printf("TCP client %s %s stopped\n", tcpClient.ip, tcpClient.port)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := tcpClient.Start(); err != nil {
				fmt.Printf("TCP client %s %s error: %v\n", tcpClient.ip, tcpClient.port, err)
			}

			select {
			case <-time.After(3 * time.Second):
				// Продолжаем работу
			case <-ctx.Done():
				return
			}
		}
	}
}

// Вариант с reconnect
// Start подключается к TCP-серверу и управляет соединением.
// func (tcpClient *TcpClient) Start() {
// 	serverAddress := tcpClient.ip + ":" + tcpClient.port
// 	var conn net.Conn
// 	var err error
// 	d := net.Dialer{Timeout: time.Duration(3) * time.Second}
// 	for {
// 		// fmt.Println("tcpClient.reconnect", tcpClient.reconnect)
// 		if !tcpClient.reconnect {
// 			return
// 		}
// 		conn, err = d.Dial("tcp", serverAddress)
// 		if err != nil {
// 			fmt.Printf("Ошибка подключения: %v. Повторная попытка через 5 секунд...\n", err)
// 		} else {
// 			fmt.Println("Подключено к серверу")
// 			tcpClient.Conn = conn

// 			// Канал для уведомления о разрыве соединения
// 			disconnectChan := make(chan struct{})

// 			// Запускаем горутину для чтения данных от сервера
// 			go handleConnection(conn, tcpClient, disconnectChan)

// 			// Ожидаем разрыва соединения
// 			<-disconnectChan
// 			if tcpClient.reconnect {
// 				fmt.Println("Соединение разорвано. Повторная попытка через 5 секунд...")
// 			} else {
// 				fmt.Println("Соединение разорвано. Повторных попыток не будет")
// 			}
// 		}
// 		time.Sleep(5 * time.Second)
// 	}
// }

// Start подключается к TCP-серверу и управляет соединением.
// вариант без reconnect
func (tcpClient *TcpClient) Start() error {
	serverAddress := tcpClient.ip + ":" + tcpClient.port
	var err error

	d := net.Dialer{Timeout: time.Duration(3) * time.Second}
	conn, err := d.Dial("tcp", serverAddress)
	if err != nil {
		fmt.Printf("Ошибка подключения: %v\n", err)
		if tcpClient.OnClose != nil {
			tcpClient.OnClose() // Уведомляем о невозможности подключения
		}
		time.Sleep(time.Duration(tcpClient.reconnectDelay) * time.Second)
		return err
	}

	fmt.Println("Подключено к серверу")
	tcpClient.Conn = conn

	// Канал для уведомления о разрыве соединения
	disconnectChan := make(chan struct{})

	if tcpClient.OnConnect != nil {
		tcpClient.OnConnect()
	}
	// Запускаем горутину для чтения данных от сервера
	go handleConnection(conn, tcpClient, disconnectChan)
	// Ожидаем разрыва соединения
	<-disconnectChan

	if tcpClient.OnClose != nil {
		tcpClient.OnClose() // Уведомляем о закрытии соединения
	}
	fmt.Println("Соединение разорвано. Завершение работы.")
	time.Sleep(time.Duration(tcpClient.reconnectDelay) * time.Second)
	return nil
}

func (tcpClient *TcpClient) Stop() {
	if tcpClient.Conn != nil {
		tcpClient.reconnect = false
		fmt.Println("tcpClient остановлен", tcpClient.ip, tcpClient.port)
		tcpClient.Conn.Close()
	}
}

// handleConnection обрабатывает входящие данные от TCP-сервера.
func handleConnection(conn net.Conn, tcpClient *TcpClient, disconnectChan chan struct{}) {
	defer func() {
		conn.Close()
		close(disconnectChan) // Уведомляем о разрыве соединения
	}()

	reader := bufio.NewReader(conn)

	for {
		message, err := reader.ReadString(10)
		if err != nil {
			fmt.Printf("Ошибка чтения: %v\n", err)
			return
		}
		fmt.Print("Получено от TCP сервера: ", message)
		if tcpClient.OnMessage != nil {
			tcpClient.OnMessage([]byte(message))
		}
	}
}

// Send отправляет данные на TCP-сервер.
func (tcpClient *TcpClient) Send(msg []byte) (err error) {
	if tcpClient.Conn != nil {
		_, err = tcpClient.Conn.Write(msg)
	} else {
		return nil
	}
	return
}
