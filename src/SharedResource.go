package main

import "net"

func main() {
    Address, err := net.ResolveUDPAddr("udp", ":10001")
    if err != nil{
        panic(err)
    }
    Connection, err := net.ListenUDP("udp", Address)
    if err != nil{
        panic(err)
    }
    defer Connection.Close()
    for {
        //todo
    }
}
