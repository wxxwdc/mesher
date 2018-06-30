package resolver

import (
	"errors"
	"log"
	"net/url"

	"github.com/ServiceComb/go-chassis/core/lager"
	"github.com/ServiceComb/go-chassis/core/util/string"
	"github.com/go-chassis/mesher/config"
)

var dr DestinationResolver
var DestinationResolverPlugins map[string]func() DestinationResolver
var SelfEndpoint = "#To be init#"

const DefaultPlugin = "host"

var ErrUnknownResolver = errors.New("unknown Destination Resolver")

type DestinationResolver interface {
	Resolve(sourceAddr string, header map[string]string, rawUri string, destinationName *string) error
}

type DefaultDestinationResolver struct {
}

func (dr *DefaultDestinationResolver) Resolve(sourceAddr string, header map[string]string, rawUri string, destinationName *string) error {
	u, err := url.Parse(rawUri)
	if err != nil {
		lager.Logger.Error("Can not parse url", err)
		return err
	}

	if u.Host == "" {
		return errors.New(`Invalid uri, please check:
1, For provider, mesher listens on external ip
2, Set http_proxy as mesher address, before sending request`)
	}

	if u.Host == SelfEndpoint {
		return errors.New(`uri format must be: http://serviceName/api`)
	}

	if h := stringutil.SplitFirstSep(u.Host, ":"); h != "" {
		*destinationName = h
		return nil
	}

	*destinationName = u.Host
	return nil
}
func New() DestinationResolver {
	return &DefaultDestinationResolver{}
}
func GetDestinationResolver() DestinationResolver {
	return dr
}
func InstallDestinationResolver(name string, newFunc func() DestinationResolver) {
	DestinationResolverPlugins[name] = newFunc
	log.Printf("Installed DestinationResolver Plugin, name=%s", name)
}
func init() {
	DestinationResolverPlugins = make(map[string]func() DestinationResolver)
	dr = &DefaultDestinationResolver{}
	InstallDestinationResolver(DefaultPlugin, New)
}
func Init() error {
	var name string
	if config.GetConfig().Plugin != nil {
		name = config.GetConfig().Plugin.DestinationResolver
	}
	if name == "" {
		name = DefaultPlugin
	}
	df, ok := DestinationResolverPlugins[name]
	if !ok {
		return ErrUnknownResolver
	}
	dr = df()
	return nil
}
