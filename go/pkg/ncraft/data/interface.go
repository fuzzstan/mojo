package data

import (
    "github.com/mojo-lang/lang/go/pkg/mojo/lang"
    "strings"
)

type Interface struct {
    Decl       *lang.InterfaceDecl
    Name       string
    BaredName  string // remove the Service postfix if the interface name is ends with 'Service'
    ServerName string

    Methods []*Method
}

func GetInterfaceServerName(name string) string {
    if strings.HasSuffix(name, "Service") {
        return strings.TrimSuffix(name, "Service") + "Server"
    } else if strings.HasSuffix(name, "Server") {
        return name
    } else {
        return name + "Server"
    }
}

func GetInterfaceBaredName(name string) string {
    name = strings.TrimSuffix(name, "Service")
    return strings.TrimSuffix(name, "Server")
}
