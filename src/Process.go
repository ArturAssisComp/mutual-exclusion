package main

import (
	"bufio"
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
    "sync"
)

// Variáveis globais interessantes para o processo
var err string
var myPort string          //porta do meu servidor
var nServers int           //qtde de outros processo
var CliConn []*net.UDPConn //vetor com conexões para os servidores
// dos outros processos
var ServConn *net.UDPConn //conexão do meu servidor (onde recebo
// mensagens dos outros processos)
var ti int           //clock
var pi int           //Process_ID
var state int        //Process_state
var todos_reply bool //indica se o processo se recebeu todos os replies
var SharedResource *net.UDPConn
var queue []int            //lista com os processos empilhados
var process_replies []int  //lista com os processos que já enviaram reply
var reply_ja_recebido bool //indica se o reply de certo processo já foi recebido

// Função enum para definir os estados possíveis
const (
	//iota começa como zero
	RELEASED = iota //Saiu da CS
	WANTED          //Esperando para entrar na CS
	HELD            //Está na CS
)

func doServerJob(mutex *sync.Mutex) {
	//Responsável por sempre receber mensagens
	buf := make([]byte, 1024)

	for {
		//LOOP INFINITO
		n, _, err := ServConn.ReadFromUDP(buf)
		msg := string(buf[0:n])
		//Fazer o splitting da mensagem
		msg_recebida := strings.Split(msg, ",")
		//pj := msg_recebida[0]
		//tj := msg_recebida[1]
		order := msg_recebida[2]

		//Converter (pj,tj) de string para int
		pj, _ := strconv.Atoi(msg_recebida[0])
		tj, _ := strconv.Atoi(msg_recebida[1])
        fmt.Printf("Recebido %s de <p:%d,t:%d>\n", order, pj, tj)

		//Ricart-Agrawala: se order for um request
		if order == "request" {
			if state == HELD || (state == WANTED && priority_check(pj, tj)) {
				//coloca o processo Pj na fila de replies
                mutex.Lock()
				queue = append(queue, pj)
                mutex.Unlock()
                fmt.Printf("Colocado na fila: <p:%d,t:%d>\n", pj, tj)
                //Update clock:
                mutex.Lock()
                oldTi := ti
                ti = int(math.Max(float64(tj), float64(ti))) + 1
                fmt.Printf("(atualizando timestamp: %d -> %d)\n", oldTi, ti)
                mutex.Unlock()
			} else {
                //Update clock:
                mutex.Lock()
                oldTi := ti
                ti = int(math.Max(float64(tj), float64(ti))) + 1
                fmt.Printf("(atualizando timestamp: %d -> %d)\n", oldTi, ti)
                mutex.Unlock()
				//Pi envia reply imediato para Pj
				//converter de int para string
				clock := strconv.Itoa(ti)
				p_id := strconv.Itoa(pi)
				//msg_reply = pi,ti,reply
				msg_reply := p_id + "," + clock + "," + "reply"
				buf := []byte(msg_reply)

				//envia para Pj
				//lembrando que a lista de conexões começa em 0
				idx := pj - 1
				_, err := CliConn[idx].Write(buf)
                fmt.Printf("Reply enviado para: <p:%d>\n", pj)
				Print_panic(err)
			}
		} else if order == "reply" {
            //Update clock:
            mutex.Lock()
            oldTi := ti
            ti = int(math.Max(float64(tj), float64(ti))) + 1
            fmt.Printf("(atualizando timestamp: %d -> %d)\n", oldTi, ti)
            mutex.Unlock()
			//verificar se este reply já recebido
			for _, p_id := range process_replies {
				if p_id == pj {
					reply_ja_recebido = true
                    break
				} else {
					reply_ja_recebido = false
				}
			}
			if !reply_ja_recebido {
				//ainda não recebeu o reply de Pj
                mutex.Lock()
				process_replies = append(process_replies, pj)
                mutex.Unlock()
			} else {
			}

			//verificar se todos os replies já foram recebidos
			if len(process_replies) == nServers - 1 {
                mutex.Lock()
				todos_reply = true
                mutex.Unlock()
			}
		} else {
			fmt.Printf("%s não é nem reply nem request: unknown msg", order)
		}
		Print_panic(err)
	}
}

func doClientJob(pj int, x string, mutex *sync.Mutex) {
	//Verificar se houve ação interna
	if pj == pi {
		//incrementa o clock
        mutex.Lock()
		ti += 1
        fmt.Printf("(atualizando timestamp: %d -> %d)\n", ti - 1, ti)
        mutex.Unlock()
	} else {
		//Ricart-Agrawala: verificar se foi solicitado acesso à CS
		if x == "x" {
			//Verificar se x é indevido
			if state == HELD || state == WANTED {
				fmt.Println("x ingnorado")
			} else {
				texto := "Oi"
				//Ricart-Agrawala: processo (pi.ti) solicita acesso
                mutex.Lock()
				state = WANTED
				//Converter int para string a fim de transmitir uma msg
				//Incrementa o clock apenas uma ve antes de enviar os requests
				ti += 1
                fmt.Printf("(atualizando timestamp: %d -> %d)\n", ti - 1, ti)
                mutex.Unlock()
				clock := strconv.Itoa(ti)
				p_id := strconv.Itoa(pi)
				//msg_acesso = pi,ti,request
				msg_acesso := p_id + "," + clock + "," + "request"
				buf := []byte(msg_acesso)

				//Multicast para todos os processos
				//Iterar na lista de processos
				//utilizar for range: retorna -> índice na lista, elemento da lista
				for i, Conn := range CliConn {
                    //subtrair de 1 pois o vetor de conexões começa com 0
                    if i != pi - 1{
                        //Conn = CliConn[i]
                        _, err := Conn.Write(buf)
                        Print_panic(err)
                        fmt.Printf("Request enviado para: <p:%d>\n", i + 1)
                    }
				}

				//Espera até receber todos os replies
				//LOOP INFINITO enquanto a condição não é atendida
				for !todos_reply {
				}

				//Se recebeu todos os replies, entrar na CS
				Usar_a_CS(pi, ti, texto, mutex)

				//Depois de usar a CS, dar release e sair
				fmt.Println("Sai da CS")
                mutex.Lock()
				ti += 1
                fmt.Printf("(atualizando timestamp: %d -> %d)\n", ti - 1, ti)
				state = RELEASED
				todos_reply = false
                mutex.Unlock()
				//Dar reply para os processos empilhados
				reply_to_queue(mutex)
				//Esvaziar a list de processos que já enviaram reply
                mutex.Lock()
				process_replies = nil
                mutex.Unlock()
			}
		}
	}
}

func initConnections() {
	//No inicio o clock é setado como zero
	ti = 0
    fmt.Printf("(iniciando timestamp: %d)\n", ti)

	pi, _ = strconv.Atoi(os.Args[1])
	myPort = os.Args[pi+1]
	nServers = len(os.Args) - 2
	/*Esse 2 tira o nome (no caso Process) e tira a primeira porta (que
	é a minha). As demais portas são dos outros processos*/
	CliConn = make([]*net.UDPConn, nServers)

	//Outros códigos para deixar ok a conexão do meu servidor (onde recebo msgs). O processo já deve ficar habilitado a receber msgs.

	//Criando um socket para receber ordens externas
	//Prepara um endereço na porta determinada
	ServerAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1"+myPort)
	panic(err)
	//Recebe na porta determinada
	ServConn, err = net.ListenUDP("udp", ServerAddr)
	panic(err)

	//Outros códigos para deixar ok as conexões com os servidores dos outros processos. Colocar tais conexões no vetor CliConn.
	for servidores := 0; servidores < nServers; servidores++ {
		//criando sockets transmissores de ordens
		ServerAddr, err := net.ResolveUDPAddr("udp",
			"127.0.0.1"+os.Args[2+servidores])
		panic(err)
		Conn, err := net.DialUDP("udp", nil, ServerAddr)
		CliConn[servidores] = Conn
		panic(err)
	}

	//Conectando com o sharedResource na porta 10001
	ServerAddr, err = net.ResolveUDPAddr("udp", "127.0.0.1:10001")
	panic(err)
	SharedResource, err = net.DialUDP("udp", nil, ServerAddr)
	panic(err)
}

func main() {
	initConnections()
	state = RELEASED
	todos_reply = false
    var mutex sync.Mutex

	//O fechamento de conexões deve ficar aqui, assim só fecha
	//conexão quando a main morrer
	defer ServConn.Close()
	for i := 0; i < nServers; i++ {
		defer CliConn[i].Close()
	}

	//Todo Process fará a mesma coisa: verificar se pode enntrar na CS eficar ouvindo mensagens

	ch := make(chan string) //canal que guarda itens lidos do teclado
	go readInput(ch)        //chamar rotina que ”escuta” o teclado
	//Responsável por receber msgs: replies ou requests
	go doServerJob(&mutex)

	for {
		//Loop Infinito

		// Verificar (de forma não bloqueante) se tem algo no
		// stdin (input do terminal)
		select {
		case x, valid := <-ch:
			if valid {
				//transformação string para int
				pj, _ := strconv.Atoi(x)
				go doClientJob(pj, x, &mutex)
			} else {
				fmt.Println("Canal fechado!")
			}
		default:
			// Fazer nada...
			// Mas não fica bloqueado esperando o teclado
			time.Sleep(time.Second * 1)
		}
		// Esperar um pouco
		time.Sleep(time.Second * 1)
	}
}

//Funções Auxiliares

// Analisa quem se Pi tem a prioridade
func priority_check(pj int, tj int) bool {
	if ti < tj {
		//Pi tem a prioridade
		return true
	} else if ti == tj {
		//desempate no process_id
		if pi < pj {
			//Pi tem prioridade
			return true
		} else {
			//Pi > Pj
			return false
		}
	} else {
		//ti > tj
		return false
	}
}

// Função que reproduz o que ocorre quando o processo entra na CS
// Enviar msg para o SharedResource e dormir um pouco
func Usar_a_CS(pi int, ti int, texto string, mutex *sync.Mutex) {
	fmt.Println("Entrei na CS")
    mutex.Lock()
	state = HELD
    mutex.Unlock()

	//converter int para string a fim de transmitir a msg
	p_id := strconv.Itoa(pi)
	clock := strconv.Itoa(ti)
	//msg_shared = pi,ti,texto
	msg_shared := p_id + "," + clock + "," + texto
	buf := []byte(msg_shared)

	//Enviar
	_, err := SharedResource.Write(buf)
	Print_panic(err)

	//Dormir...
	time.Sleep(time.Second * 2)
}

// Envia msgs de reply para todos os processos empilhados
func reply_to_queue(mutex *sync.Mutex) {
	//Para enviar mensagens de reply
	//converter de int para string
	clock := strconv.Itoa(ti)
	p_id := strconv.Itoa(pi)
	//msg_reply = pi,ti,reply
	msg_reply := p_id + "," + clock + "," + "reply"
	buf := []byte(msg_reply)

	//Enviar reply para todos os processos empilhados
	//Reply não incrementa clock
	for _, pj := range queue {
		//subtrair de 1 pois o vetor de conexões começa com 0
		idx := pj - 1
		//Enviar reply
		_, err := CliConn[idx].Write(buf)
        fmt.Printf("Reply enviado para (da fila): <p:%d>\n", pj)
		Print_panic(err)
	}
    //Reset queue:
    mutex.Lock()
    queue = make([]int, 0)
    mutex.Unlock()
}

// Referência: CES27-AtividadeDirigida-LogicalClock.pdf
// Checa se existe erro
func panic(err error) {
	if err != nil {
		fmt.Println("Erro: ", err)
		os.Exit(0)
	}
}
func Print_panic(err error) {
	if err != nil {
		fmt.Println("Erro: ", err)
	}
}

// Lê input do teclado
func readInput(ch chan string) {
	// Rotina não-bloqueante que “escuta” o stdin
	reader := bufio.NewReader(os.Stdin)
	for {
		text, _, _ := reader.ReadLine()
		ch <- string(text)
	}
}
