package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

// Variáveis globais interessantes para o processo
var err string
var myPort string          //porta do meu servidor
var nServers int           //qtde de outros processo
var CliConn []*net.UDPConn //vetor com conexões para os servidores
// dos outros processos
var ServConn *net.UDPConn //conexão do meu servidor (onde recebo
// mensagens dos outros processos)
var clock int
var id int

// Referência: CES27-AtividadeDirigida-LogicalClock.pdf
func panic(err error) {
	if err != nil {
		fmt.Println("Erro: ", err)
		os.Exit(0)
	}
}

func main() {
	Address, err := net.ResolveUDPAddr("udp", ":10001")
	panic(err)
	Connection, err := net.ListenUDP("udp", Address)
	panic(err)
	defer Connection.Close()

	buf := make([]byte, 1024)

	for {
		//Loop infinito para receber mensagem e escrever todo
		//conteúdo (processo que enviou, relógio recebido e texto)
		//na tela

		//A entrada será na formato:
		/* Ti,Pi,request
		   Tj,Pj,release */

		n, _, err := ServConn.ReadFromUDP(buf)
		msg := string(buf[0:n])

		//Uma vez recebida a entrada, separar a string pela vírgula
		msg_print := strings.Split(msg, ",")

		//Novo formato:
		//msg_print[0] = "Ti"
		//msg_print[1] = "Pi"
		//msg_print[2] = "mensagem"
		//Imprimir a entrada na tela: (Ti, Pi, msg)
		fmt.Printf("<Clock, Process_ID, Message>: (%s, %s, %s)\n", msg_print[0], msg_print[1], msg_print[2])

		//Se houver erro
		if err != nil {
			fmt.Println("Erro: ", err)
		}
	}
}
