package main

import (
	"fmt"
	"os"

	. "github.com/aws/aws-lambda-go/lambda"
	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payment-reconciler/config"
	"github.com/companieshouse/payment-reconciler/lambda"
)

func main() {

	log.Namespace = "payment-reconciler"

	cfg, err := config.Get()
	if err != nil {
		log.Error(fmt.Errorf("error configuring service: %s. - exiting", err), nil)
		return
	}

	log.Trace("Config", log.Data{"Config": cfg})
	log.Info("Payment reconciliation lambda started")

	reconciliationLambda, err := lambda.New(cfg)
	if err != nil {
		log.Error(fmt.Errorf("error initializing lambda: %s", err), nil)
		os.Exit(1)
	}

	Start(reconciliationLambda.Execute)
}
