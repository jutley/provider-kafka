package topic

import (
	"context"
	//"github.com/crossplane/crossplane-runtime/pkg/meta"
	//"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	//"github.com/twmb/franz-go/pkg/kerr"

	"github.com/pkg/errors"
	"github.com/twmb/franz-go/pkg/kadm"

	"github.com/crossplane-contrib/provider-kafka/apis/topic/v1alpha1"
)

// Topic is a holistic representation of a Kafka Topic with all configurable
// fields
type Topic struct {
	Name              string
	ReplicationFactor int16
	Partitions        int32
	ID                string
	Config            map[string]string
}

// Get gets the topic from Kafka side and returns a Topic object.
func Get(ctx context.Context, client *kadm.Client, name string) (*Topic, error) {
	// TODO: We first need to get the topic (via ListTopics) to fill the ID,
	//  ReplicationFactor and Partitions fields. Then call DescribeTopicConfigs
	//  to fill Topic.Config.

	td, err := client.ListTopics(ctx, name)
	if err != nil {
		return nil, errors.Wrap(err, "cannot list topics")
	}
	if td[name].Err != nil {
		return nil, errors.Wrap(td[name].Err, "topic does not exist in kafka cluster")
	}

	t, ok := td[name]
	if !ok {
		return nil, errors.New("no create response for topic")
	}

	tc, err := client.DescribeTopicConfigs(ctx, name)
	if err != nil {
		return nil, errors.Wrap(err, "cannot describe topics")
	}

	ts := Topic{}
	ts.Name = name
	ts.ReplicationFactor = int16(t.Partitions[0].Replicas[0])
	ts.Partitions = t.Partitions[0].Partition
	ts.ID = name
	ts.Config = make(map[string]string)

	for _, value := range tc[0].Configs {
		ts.Config[value.Key] = *value.Value
	}
	return &ts, nil

}

func Create(ctx context.Context, client *kadm.Client, topic *Topic) error {
	// TODO: Call client.CreateTopics using provided Topic
	resp, err := client.CreateTopics(ctx, topic.Partitions, topic.ReplicationFactor, nil, topic.Name)
	if err != nil {
		return err
	}

	t, ok := resp[topic.Name]
	if !ok {
		return errors.New("no create response for topic")
	}
	if t.Err != nil {
		return errors.Wrap(t.Err, "cannot create topic")
	}

	// TODO: Could we pass topic configs in Create call? Otherwise, also call
	//  AlterConfig for each config provided.
	return nil
}

// Delete deletes the topic from Kafka side
func Delete(ctx context.Context, client *kadm.Client, name string) error {

	td, err := client.DeleteTopics(ctx, name)
	if err != nil {
		return err
	}

	t, ok := td[name]
	if !ok {
		return errors.New("no delete response for topic")
	}
	if t.Err != nil {
		return errors.Wrap(t.Err, "cannot delete topic")
	}

	return nil
}

func Update(ctx context.Context, client *kadm.Client, desired *Topic) error {
	// First Get existing Topic
	existing, err := Get(ctx, client, desired.Name)
	if err != nil {
		return errors.Wrap(err, "cannot get topic")
	}
	if existing == nil {
		return errors.New("topic does not exist")
	}
	// TODO: Update if Partitions needs to be updated

	// TODO: Update all configs as in the spec. Yes, we might call an Update
	//  (i.e. Set), also for the ones didn't change, but this shouldn't be
	//  a problem, given we do that only when something is not up to date.
	return nil
}

// topic.Generate(meta.GetExternalName(cr), &cr.Spec.ForProvider)
// Generate is used to convert Crossplane TopicParameters to Kafka's Topic.
func Generate(name string, params *v1alpha1.TopicParameters) *Topic {
	tpc := &Topic{
		Name:              name,
		ReplicationFactor: int16(params.ReplicationFactor),
		Partitions:        int32(params.Partitions),
	}

	if len(params.Config) > 0 {
		tpc.Config = make(map[string]string, len(params.Config))
		for k, v := range params.Config {
			tpc.Config[k] = v
		}
	} else {
		tpc.Config = nil
	}

	return tpc
}

// LateInitializeSpec fills empty spec fields with the data retrieved from Kafka.
func LateInitializeSpec(params *v1alpha1.TopicParameters, observed *Topic) bool {
	lateInitialized := false
	if params.Config == nil {
		params.Config = make(map[string]string, len(observed.Config))
	}

	for k, v := range observed.Config {
		if _, ok := params.Config[k]; !ok {
			lateInitialized = true
			params.Config[k] = v
		}
	}
	return lateInitialized
}

// IsUpToDate returns true if the supplied Kubernetes resource differs from the
// supplied Kafka Topic.
func IsUpToDate(in *v1alpha1.TopicParameters, observed *Topic) bool {
	if in.Partitions != int(observed.Partitions) {
		return false
	}
	if in.ReplicationFactor != int(observed.ReplicationFactor) {
		return false
	}
	if len(in.Config) != len(observed.Config) {
		return false
	}
	for k, v := range observed.Config {
		if iv, ok := in.Config[k]; !ok || iv != v {
			return false
		}
	}
	return true
}
