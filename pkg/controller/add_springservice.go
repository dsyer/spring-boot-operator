package controller

import (
	"github.com/dsyer/spring-boot-operator/pkg/controller/springservice"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, springservice.Add)
}
