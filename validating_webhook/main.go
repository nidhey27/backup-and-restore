package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/spf13/pflag"
	"k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/options"
	"k8s.io/component-base/cli/globalflag"
)

const (
	valCon = "val-controller"
)

type Options struct {
	SecureServingOptions options.SecureServingOptions
}

// Method on Option struct to add flags
func (o *Options) AddFlagSet(fs *pflag.FlagSet) {
	o.SecureServingOptions.AddFlags(fs)
}

// Method to return config that will eventually run server

type Config struct {
	SecureServingInfo *server.SecureServingInfo
}

func (o *Options) Config() *Config {
	if err := o.SecureServingOptions.MaybeDefaultWithSelfSignedCerts("0.0.0.0", nil, nil); err != nil {
		panic(err)
	}
	c := &Config{}

	o.SecureServingOptions.ApplyTo(&c.SecureServingInfo)
	return c
}

// Func to return default options
func NewDefaultOptions() *Options {
	o := &Options{
		SecureServingOptions: *options.NewSecureServingOptions(),
	}

	o.SecureServingOptions.BindPort = 8443
	// If we will not provide crt and key file then this will generate those files for us
	o.SecureServingOptions.ServerCert.PairName = valCon

	return o
}

func main() {
	options := NewDefaultOptions()
	fs := pflag.NewFlagSet(valCon, pflag.ExitOnError)

	// Add -help flag for our app
	globalflag.AddGlobalFlags(fs, valCon)

	// Adding
	options.AddFlagSet(fs)

	// Parse flag set
	err := fs.Parse(os.Args)

	if err != nil {
		log.Printf("ERROR: %s", err.Error())
	}

	c := options.Config()

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(ServeCRValidation))

	stopCh := server.SetupSignalHandler()

	_, ch, err := c.SecureServingInfo.Serve(mux, 30*time.Second, stopCh)

	if err != nil {
		log.Printf("ERROR: %s", err.Error())
	} else {
		<-ch
	}
}

func ServeCRValidation(w http.ResponseWriter, r *http.Request) {
	fmt.Println("ServeCRValidation was called")
}
