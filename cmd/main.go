package main

import (
	"log"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/service"
	"go.opentelemetry.io/collector/service/defaultcomponents"

	"github.com/koelschkellerkind/owtraceprocessor"
)

func main() {
	factories, err := components()
	if err != nil {
		log.Fatalf("failed to build components: %v", err)
	}

	info := component.ApplicationStartInfo{
		ExeName:  "owtracecollector",
		LongName: "Collector to enrich traces coming from OpenWhisk",
		Version:  "1.0.0",
	}

	app, err := service.New(service.Parameters{ApplicationStartInfo: info, Factories: factories})
	if err != nil {
		log.Fatal("failed to construct the application: %w", err)
	}

	err = app.Run()
	if err != nil {
		log.Fatal("application run finished with error: %w", err)
	}
}

func components() (component.Factories, error) {
	var errs []error
	factories, err := defaultcomponents.Components()
	if err != nil {
		return component.Factories{}, err
	}
	processors := []component.ProcessorFactory{
		owtraceprocessor.NewFactory(),
	}
	for _, pr := range factories.Processors {
		processors = append(processors, pr)
	}
	factories.Processors, err = component.MakeProcessorFactoryMap(processors...)
	if err != nil {
		errs = append(errs, err)
	}

	return factories, componenterror.CombineErrors(errs)
}
