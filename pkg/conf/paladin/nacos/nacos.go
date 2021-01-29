package nacos

import (
	"context"
	"flag"
	"github.com/go-kratos/kratos/pkg/conf/env"
	"github.com/go-kratos/kratos/pkg/conf/paladin"
	"github.com/go-kratos/kratos/pkg/log"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/pkg/errors"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
)

const (
	PaladinDriverNacos = "nacos"
	defaultChSize      = 10
)

var (
	nacosTimeoutMs           uint64
	nacosNotLoadCacheAtStart bool
	nacosLogDir              string
	nacosCacheDir            string
	nacosServer              string
	nacosNamespace           string
	nacosLogLevel            string
)

func init() {
	paladin.Register(PaladinDriverNacos, &nacosDriver{})
	flag.StringVar(&nacosLogDir, "nacos.logdir", os.Getenv("NACOS_LOGDIR"), "nacos log directory")
	flag.StringVar(&nacosCacheDir, "nacos.cachedir", os.Getenv("NACOS_CACHEDIR"), "nacos cache directory")
	flag.StringVar(&nacosServer, "nacos.servers", os.Getenv("NACOS_SERVERS"), "nacos servers")
	flag.StringVar(&nacosNamespace, "nacos.namespace", os.Getenv("NACOS_NAMESPACE"), "nacos namespace")
	flag.BoolVar(&nacosNotLoadCacheAtStart, "nacos.nocache", false, "nacos not load cache at start,default : false")
	flag.Uint64Var(&nacosTimeoutMs, "nacos.timeout", 5000, "nacos timeout , default 5000 ms")
	flag.StringVar(&nacosLogLevel, "nacos.loglevel", defaultString("NACOS_LOGLEVEL", "info"), "the level of log, it's must be debug,info,warn,error, default info")
}

type nacosDriver struct{}

func (nd *nacosDriver) New() (paladin.Client, error) {

	clientConfig, serverConfigs, err := buildNacosConfig()
	if err != nil {
		return nil, err
	}

	return nd.new(clientConfig, serverConfigs)

}
func defaultString(env, value string) string {
	v := os.Getenv(env)
	if v == "" {
		return value
	}
	return v
}
func buildNacosConfig() (*constant.ClientConfig, []constant.ServerConfig, error) {
	if env.AppID == "" {
		return nil, nil, errors.New("appid is null, use appid or APP_ID env.")
	}
	if nacosNamespace == "" {
		return nil, nil, errors.New("invalid nacos namespace")
	}
	if nacosLogDir == "" {
		return nil, nil, errors.New("invalid nacos log dir")
	}
	if nacosCacheDir == "" {
		return nil, nil, errors.New("invalid nacos cache dir")
	}
	if nacosServer == "" {
		return nil, nil, errors.New("invalid nacos servers")
	}

	clientConfig := &constant.ClientConfig{
		NamespaceId:         nacosNamespace,
		TimeoutMs:           nacosTimeoutMs,
		NotLoadCacheAtStart: nacosNotLoadCacheAtStart,
		LogDir:              nacosLogDir,
		CacheDir:            nacosCacheDir,
		LogLevel:            nacosLogLevel,
	}

	servers := strings.Split(nacosServer, ",")
	serverConfigs := make([]constant.ServerConfig, 0)
	for _, server := range servers {
		u, err := url.Parse(server)
		if err != nil {
			return nil, nil, errors.New("nacos servers parse error ")
		}
		port, _ := strconv.Atoi(u.Port())
		serverConfig := constant.ServerConfig{
			Scheme:      u.Scheme,
			ContextPath: u.Path,
			IpAddr:      u.Hostname(),
			Port:        uint64(port),
		}
		serverConfigs = append(serverConfigs, serverConfig)
	}
	if len(serverConfigs) == 0 {
		return nil, nil, errors.New("nacos servers parse error ")
	}
	return clientConfig, serverConfigs, nil
}

func (nd *nacosDriver) new(clientConfig *constant.ClientConfig, serverConfigs []constant.ServerConfig) (paladin.Client, error) {
	client, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		return nil, err
	}

	n := &nacosClient{
		appID:       env.AppID,
		client:      client,
		groupValues: new(paladin.Map),
		lock:        new(sync.RWMutex),
		ids:         make([]string, 0),
		chans:       make(map[string][]chan paladin.Event),
		rawVal:      make(map[string]*paladin.Value),
	}
	return n, nil
}

type nacosClient struct {
	appID       string
	client      config_client.IConfigClient
	rawVal      map[string]*paladin.Value
	groupValues *paladin.Map
	lock        *sync.RWMutex
	ids         []string
	chans       map[string][]chan paladin.Event
}

// Get a config value by a config key(may be a sven filename).
func (n *nacosClient) Get(key string) *paladin.Value {
	v, err := n.loadValue(key)
	if err != nil {
		log.Error("[nacosDriver] get config nacod ID %s error: %+v", key, err)
		return paladin.NewValue(nil, "")
	}
	return v
}

// GetAll nacos listen use [dataId+group] ,so before getall must call Watch or WatchEvent
func (n *nacosClient) GetAll() *paladin.Map {
	return n.groupValues
}

func (n *nacosClient) WatchEvent(ctx context.Context, nacosIDs ...string) <-chan paladin.Event {
	n.lock.Lock()
	defer n.lock.Unlock()
	ch := make(chan paladin.Event, defaultChSize)
	for _, key := range nacosIDs {
		err := n.client.ListenConfig(vo.ConfigParam{
			DataId: key,
			Group:  n.appID,
			OnChange: func(namespace, group, dataId, data string) {
				n.reloadValue(dataId, data)
			},
		})
		if err != nil {
			log.Errorc(ctx, "[nacosDriver] listen config error %+v", err)
			continue
		}
		n.ids = append(n.ids, key)

		n.chans[key] = append(n.chans[key], ch)
	}

	return ch
}

// Close is a WatchClose func
func (n *nacosClient) Close() error {
	n.lock.Lock()
	defer n.lock.Unlock()
	for _, id := range n.ids {
		err := n.client.CancelListenConfig(vo.ConfigParam{
			DataId: id,
			Group:  n.appID,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (n *nacosClient) loadValue(dataId string) (*paladin.Value, error) {
	content, err := n.client.GetConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  n.appID,
	})
	if err != nil {
		return nil, err
	}
	val := paladin.NewValue(content, content)
	n.rawVal[dataId] = val
	n.groupValues.Store(n.rawVal)
	return val, nil
}
func (n *nacosClient) reloadValue(dataId string, data string) {
	_, err := n.loadValue(dataId)
	if err != nil {
		log.Error("load nacos id %s error: %s, skipped", dataId, err)
		return
	}
	n.lock.Lock()

	chs := n.chans[dataId]
	defer n.lock.Unlock()

	for _, ch := range chs {
		select {
		case ch <- paladin.Event{
			Event: paladin.EventUpdate,
			Key:   dataId,
			Value: data,
		}:
		default:
			log.Info("event channel full nacos ID %s update event", dataId)
		}
	}
}
