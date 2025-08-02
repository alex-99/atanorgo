package avptcp

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"time"
)

// TcpClient is a middleman between the websocket connection and the hub.
type TcpClient struct {
	ip             string
	port           string
	reconnect      bool
	Telnet         bool
	Name           string
	HelloMessage   string
	Conn           net.Conn
	OnMessage      func(msg []byte)
	OnClose        func()
	OnConnect      func()
	reconnectDelay int
	Debug          bool
	Delim          byte
	ReadTimeout    time.Duration
}

func Version() string {
	return "1.0.4"
}

// NewTcpClient создает новый экземпляр TcpClient.
func NewTcpClient(ip string, port string) *TcpClient {
	tcpClient := &TcpClient{
		ip:             ip,
		port:           port,
		reconnectDelay: 3,
		HelloMessage:   "",
		//reconnect: true,
		Debug: false,
		// Delim:       10,
		Telnet:      false,
		ReadTimeout: 500 * time.Microsecond,
	}
	return tcpClient
}

func (tcpClient *TcpClient) StartReconnectGo(ctx context.Context) {
	go tcpClient.startReconnect(ctx)
}

func (tcpClient *TcpClient) Print(format string, a ...any) {
	if tcpClient.Debug {
		fmt.Printf(format, a...)
	}
}

// StartReconnect запускает клиента повторно после ошибки соединения
func (tcpClient *TcpClient) startReconnect(ctx context.Context) {
	tcpClient.Print("TCP client %s  %s started\n", tcpClient.ip, tcpClient.port)

	defer func() {
		tcpClient.Stop()
		tcpClient.Print("TCP client %s %s stopped\n", tcpClient.ip, tcpClient.port)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := tcpClient.Start(); err != nil {
				tcpClient.Print("TCP client %s %s error: %v\n", tcpClient.ip, tcpClient.port, err)
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
		tcpClient.Print("Ошибка подключения: %v\n", err)
		if tcpClient.OnClose != nil {
			tcpClient.OnClose() // Уведомляем о невозможности подключения
		}
		time.Sleep(time.Duration(tcpClient.reconnectDelay) * time.Second)
		return err
	}

	tcpClient.Print("Connected to server %s %s\n", tcpClient.ip, tcpClient.port)
	tcpClient.Conn = conn

	// Канал для уведомления о разрыве соединения
	disconnectChan := make(chan struct{})

	if tcpClient.OnConnect != nil {
		if tcpClient.Telnet {
			sendTelnetHandshake(conn)
		}
		if tcpClient.HelloMessage != "" {
			tcpClient.Send([]byte(tcpClient.HelloMessage))
		}
		tcpClient.OnConnect()
	}
	// Запускаем горутину для чтения данных от сервера
	go handleConnectionScan(conn, tcpClient, disconnectChan)
	// Ожидаем разрыва соединения
	<-disconnectChan

	if tcpClient.OnClose != nil {
		tcpClient.OnClose() // Уведомляем о закрытии соединения
	}
	tcpClient.Print("Connection closed. Stopping the work.")
	time.Sleep(time.Duration(tcpClient.reconnectDelay) * time.Second)
	return nil
}

func (tcpClient *TcpClient) Stop() {
	if tcpClient.Conn != nil {
		tcpClient.reconnect = false
		tcpClient.Print("tcpClient остановлен %s %s\n", tcpClient.ip, tcpClient.port)
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
		message, err := reader.ReadString(tcpClient.Delim)
		if err != nil {
			tcpClient.Print("Error reading: %v  %s %s\n", err, tcpClient.ip, tcpClient.port)
			return
		}
		tcpClient.Print("Received from TCP server: %s  %s %s\n", message, tcpClient.ip, tcpClient.port)
		if tcpClient.OnMessage != nil {
			tcpClient.OnMessage([]byte(message))
		}
	}
}

func handleConnectionScan(conn net.Conn, tcpClient *TcpClient, disconnectChan chan struct{}) {
	defer func() {
		conn.Close()
		close(disconnectChan)
	}()

	// Минимальный рабочий таймаут (не менее 10ms)
	const readTimeout = 10 * time.Millisecond
	// reader := bufio.NewReader(conn)
	buffer := make([]byte, 4096)
	partialMessage := bytes.NewBuffer(nil)

	for {
		// Важно устанавливать таймаут ПЕРЕД каждым чтением
		conn.SetReadDeadline(time.Now().Add(readTimeout))

		n, err := conn.Read(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Таймаут - проверяем накопленные данные
				if partialMessage.Len() > 0 {
					processPartialMessage(partialMessage, tcpClient)
				}
				continue
			}

			if err == io.EOF {
				// Обработка оставшихся данных при разрыве соединения
				if partialMessage.Len() > 0 {
					processPartialMessage(partialMessage, tcpClient)
				}
				return
			}

			tcpClient.Print("Read error: %v\n", err)
			return
		}

		// Обработка полученных данных
		processReceivedData(buffer[:n], tcpClient.Delim, partialMessage, tcpClient)
	}
}

func processReceivedData(data []byte, delim byte, buf *bytes.Buffer, tcpClient *TcpClient) {
	for _, b := range data {
		if b == delim {
			// Нашли разделитель - обрабатываем сообщение
			if buf.Len() > 0 {
				msg := buf.Bytes()
				tcpClient.Print("Received: %s\n", string(msg))
				if tcpClient.OnMessage != nil {
					tcpClient.OnMessage(msg)
				}
				buf.Reset()
			}
		} else {
			buf.WriteByte(b)
		}
	}
}

func processPartialMessage(buf *bytes.Buffer, tcpClient *TcpClient) {
	msg := buf.Bytes()
	tcpClient.Print("Received partial: %s\n", string(msg))
	if tcpClient.OnMessage != nil {
		tcpClient.OnMessage(msg)
	}
	buf.Reset()
}

func sendTelnetHandshake(conn net.Conn) error {
	// Базовые опции
	handshake := []byte{
		255, 253, 1, // IAC DO ECHO
		255, 253, 3, // IAC DO Suppress Go Ahead
		255, 254, 34, // IAC DONT Linemode
	}

	_, err := conn.Write(handshake)
	return err
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
