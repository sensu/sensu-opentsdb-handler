package main

import (
	"fmt"
	"net"
	"strings"
	"time"

	retry "github.com/avast/retry-go"

	"github.com/sensu-community/sensu-plugin-sdk/sensu"
	corev2 "github.com/sensu/sensu-go/api/core/v2"
)

type HandlerConfig struct {
	sensu.PluginConfig

	Host string
	Port string

	// TagHost indicates whether we should add a tag named 'host' that holds the
	// entity name.
	TagHost bool

	// Tags is a set of tags to add to the metric. They are merged with the tags
	// coming from the event's metrics, precedence being given to the latter.
	Tags map[string]string

	// PrefixEntityName indicates whether we should prefix the metric name with
	// the entity name.
	PrefixEntityName bool

	// Prefix is the prefix to add to the metric name.
	Prefix string

	// Retries is the number of attempts that should be made to connect to the
	// OpenTSDB server.
	Retries uint

	// RetryDelay is the delay in seconds we should wait between connection
	// retries.
	RetryDelay uint

	// SpaceReplacement is a string to replace potential spaces with in the
	// entity name and the event's metric tags.
	SpaceReplacement string

	conn net.Conn
}

type ConfigOptions struct {
	Host             sensu.PluginConfigOption
	Port             sensu.PluginConfigOption
	TagHost          sensu.PluginConfigOption
	Tags             sensu.PluginConfigOption
	PrefixEntityName sensu.PluginConfigOption
	Prefix           sensu.PluginConfigOption
	Retries          sensu.PluginConfigOption
	RetryDelay       sensu.PluginConfigOption
	SpaceReplacement sensu.PluginConfigOption
}

var (
	handlerConfig = HandlerConfig{
		PluginConfig: sensu.PluginConfig{
			Name:     "sensu-opentsdb-handler",
			Short:    "an opentsdb handler built for use with sensu",
			Keyspace: "sensu.io/plugins/sensu-opentsdb-handler/config",
		},
	}

	handlerConfigOptions = ConfigOptions{
		Host: sensu.PluginConfigOption{
			Path:      "host",
			Env:       "SENSU_OPENTSDB_HANDLER_HOST",
			Argument:  "host",
			Shorthand: "",
			Default:   "localhost",
			Usage:     "OpenTSDB host to send metrics to",
			Value:     &handlerConfig.Host,
		},
		Port: sensu.PluginConfigOption{
			Path:      "port",
			Env:       "SENSU_OPENTSDB_HANDLER_PORT",
			Argument:  "port",
			Shorthand: "",
			Default:   "4242",
			Usage:     "OpenTSDB port to send metrics to",
			Value:     &handlerConfig.Port,
		},
		TagHost: sensu.PluginConfigOption{
			Path:      "tag-host",
			Env:       "SENSU_OPENTSDB_HANDLER_TAGHOST",
			Argument:  "tag-host",
			Shorthand: "",
			Default:   true,
			Usage:     "Add a host tag holding the entity name to metrics",
			Value:     &handlerConfig.TagHost,
		},
		Tags: sensu.PluginConfigOption{
			Path:      "tags",
			Env:       "SENSU_OPENTSDB_HANDLER_TAGS",
			Argument:  "tags",
			Shorthand: "",
			Default:   map[string]string{},
			Usage:     "Add these tags to metrics",
			Value:     &handlerConfig.Tags,
		},
		PrefixEntityName: sensu.PluginConfigOption{
			Path:      "prefix-entity-name",
			Env:       "SENSU_OPENTSDB_HANDLER_PREFIX_ENTITY_NAME",
			Argument:  "prefix-entity-name",
			Shorthand: "",
			Default:   false,
			Usage:     "Prefix metrics name with the entity name",
			Value:     &handlerConfig.PrefixEntityName,
		},
		Prefix: sensu.PluginConfigOption{
			Path:      "prefix",
			Env:       "SENSU_OPENTSDB_HANDLER_PREFIX",
			Argument:  "prefix",
			Shorthand: "",
			Default:   "",
			Usage:     "Prefix metrics name with this string",
			Value:     &handlerConfig.Prefix,
		},
		Retries: sensu.PluginConfigOption{
			Path:      "retries",
			Env:       "SENSU_OPENTSDB_HANDLER_RETRIES",
			Argument:  "retries",
			Shorthand: "",
			Default:   uint(3),
			Usage:     "Number of times to try to connect to the server",
			Value:     &handlerConfig.Retries,
		},
		RetryDelay: sensu.PluginConfigOption{
			Path:      "retry-delay",
			Env:       "SENSU_OPENTSDB_HANDLER_RETRY_DELAY",
			Argument:  "retry-delay",
			Shorthand: "",
			Default:   uint(1),
			Usage:     "Delay in seconds between connection attempts",
			Value:     &handlerConfig.RetryDelay,
		},
		SpaceReplacement: sensu.PluginConfigOption{
			Path:      "space-replacement",
			Env:       "SENSU_OPENTSDB_HANDLER_SPACE_REPLACEMENT",
			Argument:  "space-replacement",
			Shorthand: "",
			Default:   "-",
			Usage:     "String to replace spaces with if the entity name or tags contain any",
			Value:     &handlerConfig.SpaceReplacement,
		},
	}

	options = []*sensu.PluginConfigOption{
		&handlerConfigOptions.Host,
		&handlerConfigOptions.Port,
		&handlerConfigOptions.TagHost,
		&handlerConfigOptions.Tags,
		&handlerConfigOptions.PrefixEntityName,
		&handlerConfigOptions.Prefix,
		&handlerConfigOptions.Retries,
		&handlerConfigOptions.RetryDelay,
		&handlerConfigOptions.SpaceReplacement,
	}
)

func main() {
	handler := sensu.NewGoHandler(&handlerConfig.PluginConfig, options, checkArgs, executeHandler)
	handler.Execute()
}

func checkArgs(event *corev2.Event) error {
	if !event.HasMetrics() {
		return fmt.Errorf("event does not contain metrics")
	}

	if handlerConfig.Host == "" {
		return fmt.Errorf("OpenTSDB host cannot be empty")
	}

	// Check that Prefix doesn't have any spaces in it
	if strings.Contains(handlerConfig.Prefix, " ") {
		return fmt.Errorf("prefix cannot contain spaces")
	}

	// Check that Tags do not have spaces in their key or value
	for k, v := range handlerConfig.Tags {
		if strings.Contains(k, " ") || strings.Contains(v, " ") {
			return fmt.Errorf("tags cannot contain spaces")
		}
	}

	return nil
}

func executeHandler(event *corev2.Event) error {
	if err := connect(); err != nil {
		return err
	}
	defer handlerConfig.conn.Close()

	for _, point := range event.Metrics.Points {
		fmt.Fprintf(handlerConfig.conn, MetricPointToOpenTSDBString(point, event.Entity.Name))

		// The server responds with something only if there is an error.
		// This is quite terrible for proper error handling: if we haven't
		// received anything from the server, it's either because there was no
		// error or we haven't received it yet, but we can't know for sure...
	}

	return nil
}

func connect() error {
	return retry.Do(func() error {
		conn, err := net.Dial("tcp", net.JoinHostPort(handlerConfig.Host, handlerConfig.Port))
		if err != nil {
			return err
		}

		handlerConfig.conn = conn
		return nil
	}, retry.Attempts(handlerConfig.Retries), retry.Delay(time.Duration(handlerConfig.RetryDelay)*time.Second))
}

func MetricPointToOpenTSDBString(point *corev2.MetricPoint, entityName string) string {
	var name string

	if point == nil {
		return ""
	}

	// Make sure the entity name we received does not have spaces in it.
	// If it does, transform the spaces into 'SpaceReplacement'
	entityName = strings.ReplaceAll(entityName, " ", handlerConfig.SpaceReplacement)

	// Make sure the tags we received do not have spaces in their Key or Value.
	// If they do, transform the spaces into 'SpaceReplacement'
	for _, tag := range point.Tags {
		tag.Name = strings.ReplaceAll(tag.Name, " ", handlerConfig.SpaceReplacement)
		tag.Value = strings.ReplaceAll(tag.Value, " ", handlerConfig.SpaceReplacement)
	}

	if handlerConfig.Prefix != "" {
		name = fmt.Sprintf("%s.", handlerConfig.Prefix)
	}

	if handlerConfig.PrefixEntityName {
		name = fmt.Sprintf("%s%s.", name, entityName)
	}

	name = fmt.Sprintf("%s%s", name, point.Name)

	// Merge the handler supplied tags with the event metric's tags, the latter
	// having precedence
	point.Tags = MergeMetricTags(MapToMetricTags(handlerConfig.Tags), point.Tags)

	// OpenTSDB demands there is at least 1 tag. If there are no tags, we force
	// the addition of the host tag.
	if handlerConfig.TagHost || len(point.Tags) == 0 {
		point.Tags = MergeMetricTags(point.Tags, []*corev2.MetricTag{{Name: "host", Value: entityName}})
	}

	tags := MetricTagsToKVString(point.Tags)
	return fmt.Sprintf("put %s %v %v %v\n", name, point.Timestamp, point.Value, tags)
}

// MetricTagSliceToKVString converts a slice of MetricTag into a space separated
// string of key=value
func MetricTagsToKVString(tags []*corev2.MetricTag) string {
	ss := []string{}

	for _, tag := range tags {
		ss = append(ss, fmt.Sprintf("%s=%s", tag.Name, tag.Value))
	}

	return strings.Join(ss, " ")
}

// MetricTagsToMap transforms a slice of MetricTag into an equivalent
// map[string]string
func MetricTagsToMap(tags []*corev2.MetricTag) map[string]string {
	m := map[string]string{}

	for _, mt := range tags {
		m[mt.Name] = mt.Value
	}

	return m
}

// MapToMetricTags transforms a map[string]string into an equivalent slice of
// MetricTag
func MapToMetricTags(m map[string]string) []*corev2.MetricTag {
	mt := []*corev2.MetricTag{}

	for k, v := range m {
		mt = append(mt, &corev2.MetricTag{Name: k, Value: v})
	}

	return mt
}

// MergeMetricTags merges the two given slices of MetricTag "right into left";
// that is the values in y take precedence over the values in x.
func MergeMetricTags(x, y []*corev2.MetricTag) []*corev2.MetricTag {
	m := MetricTagsToMap(x)

	for _, mt := range y {
		m[mt.Name] = mt.Value
	}

	return MapToMetricTags(m)
}
