package core

import (
	"testing"

	"github.com/projectdiscovery/nuclei/v3/pkg/cruisecontrol"
	"github.com/projectdiscovery/nuclei/v3/pkg/model/types/stringslice"
	"github.com/projectdiscovery/nuclei/v3/pkg/operators"
	"github.com/projectdiscovery/nuclei/v3/pkg/output"
	"github.com/projectdiscovery/nuclei/v3/pkg/progress"
	"github.com/projectdiscovery/nuclei/v3/pkg/protocols"
	"github.com/projectdiscovery/nuclei/v3/pkg/protocols/common/contextargs"
	"github.com/projectdiscovery/nuclei/v3/pkg/protocols/dns/dnsclientpool"
	"github.com/projectdiscovery/nuclei/v3/pkg/protocols/http/httpclientpool"
	"github.com/projectdiscovery/nuclei/v3/pkg/protocols/network/networkclientpool"
	"github.com/projectdiscovery/nuclei/v3/pkg/scan"
	"github.com/projectdiscovery/nuclei/v3/pkg/types"
	"github.com/projectdiscovery/nuclei/v3/pkg/workflows"
	"github.com/stretchr/testify/require"
)

var (
	standardProgressBar, _   = progress.NewStatsTicker(0, false, false, false, 0)
	stdOptions               = &types.Options{TemplateThreads: 10}
	standardCruiseControl, _ = cruisecontrol.New(cruisecontrol.ParseOptionsFrom(stdOptions))
	httpClientPool, _        = httpclientpool.New(stdOptions)
	dnsClientPool, _         = dnsclientpool.New(stdOptions)
	networkClientPool, _     = networkclientpool.New(stdOptions)
	execOptions              = &protocols.ExecutorOptions{
		CruiseControl:     standardCruiseControl,
		HttpClientPool:    httpClientPool,
		DnsClientPool:     dnsClientPool,
		NetworkClientPool: networkClientPool,
	}
)

func TestWorkflowsSimple(t *testing.T) {
	workflow := &workflows.Workflow{Options: execOptions, Workflows: []*workflows.WorkflowTemplate{
		{Executers: []*workflows.ProtocolExecuterPair{{
			Executer: &mockExecuter{result: true}, Options: &protocols.ExecutorOptions{Progress: standardProgressBar}},
		}},
	}}

	engine := &Engine{}
	input := contextargs.NewWithInput("https://test.com")
	ctx := scan.NewScanContext(input)
	matched := engine.executeWorkflow(ctx, workflow)
	require.True(t, matched, "could not get correct match value")
}

func TestWorkflowsSimpleMultiple(t *testing.T) {
	var firstInput, secondInput string
	workflow := &workflows.Workflow{Options: execOptions, Workflows: []*workflows.WorkflowTemplate{
		{Executers: []*workflows.ProtocolExecuterPair{{
			Executer: &mockExecuter{result: true, executeHook: func(input *contextargs.MetaInput) {
				firstInput = input.Input
			}}, Options: &protocols.ExecutorOptions{Progress: standardProgressBar}},
		}},
		{Executers: []*workflows.ProtocolExecuterPair{{
			Executer: &mockExecuter{result: true, executeHook: func(input *contextargs.MetaInput) {
				secondInput = input.Input
			}}, Options: &protocols.ExecutorOptions{Progress: standardProgressBar}},
		}},
	}}

	engine := &Engine{}
	input := contextargs.NewWithInput("https://test.com")
	ctx := scan.NewScanContext(input)
	matched := engine.executeWorkflow(ctx, workflow)
	require.True(t, matched, "could not get correct match value")

	require.Equal(t, "https://test.com", firstInput, "could not get correct first input")
	require.Equal(t, "https://test.com", secondInput, "could not get correct second input")
}

func TestWorkflowsSubtemplates(t *testing.T) {
	var firstInput, secondInput string
	workflow := &workflows.Workflow{Options: execOptions, Workflows: []*workflows.WorkflowTemplate{
		{Executers: []*workflows.ProtocolExecuterPair{{
			Executer: &mockExecuter{result: true, executeHook: func(input *contextargs.MetaInput) {
				firstInput = input.Input
			}, outputs: []*output.InternalWrappedEvent{
				{OperatorsResult: &operators.Result{}, Results: []*output.ResultEvent{{}}},
			}}, Options: &protocols.ExecutorOptions{Progress: standardProgressBar}},
		}, Subtemplates: []*workflows.WorkflowTemplate{{Executers: []*workflows.ProtocolExecuterPair{{
			Executer: &mockExecuter{result: true, executeHook: func(input *contextargs.MetaInput) {
				secondInput = input.Input
			}}, Options: &protocols.ExecutorOptions{Progress: standardProgressBar}},
		}}}},
	}}

	engine := &Engine{}
	input := contextargs.NewWithInput("https://test.com")
	ctx := scan.NewScanContext(input)
	matched := engine.executeWorkflow(ctx, workflow)
	require.True(t, matched, "could not get correct match value")

	require.Equal(t, "https://test.com", firstInput, "could not get correct first input")
	require.Equal(t, "https://test.com", secondInput, "could not get correct second input")
}

func TestWorkflowsSubtemplatesNoMatch(t *testing.T) {
	var firstInput, secondInput string
	workflow := &workflows.Workflow{Options: execOptions, Workflows: []*workflows.WorkflowTemplate{
		{Executers: []*workflows.ProtocolExecuterPair{{
			Executer: &mockExecuter{result: false, executeHook: func(input *contextargs.MetaInput) {
				firstInput = input.Input
			}}, Options: &protocols.ExecutorOptions{Progress: standardProgressBar}},
		}, Subtemplates: []*workflows.WorkflowTemplate{{Executers: []*workflows.ProtocolExecuterPair{{
			Executer: &mockExecuter{result: true, executeHook: func(input *contextargs.MetaInput) {
				secondInput = input.Input
			}}, Options: &protocols.ExecutorOptions{Progress: standardProgressBar}},
		}}}},
	}}

	engine := &Engine{}
	input := contextargs.NewWithInput("https://test.com")
	ctx := scan.NewScanContext(input)
	matched := engine.executeWorkflow(ctx, workflow)
	require.False(t, matched, "could not get correct match value")

	require.Equal(t, "https://test.com", firstInput, "could not get correct first input")
	require.Equal(t, "", secondInput, "could not get correct second input")
}

func TestWorkflowsSubtemplatesWithMatcher(t *testing.T) {
	var firstInput, secondInput string
	workflow := &workflows.Workflow{Options: execOptions, Workflows: []*workflows.WorkflowTemplate{
		{Executers: []*workflows.ProtocolExecuterPair{{
			Executer: &mockExecuter{result: true, executeHook: func(input *contextargs.MetaInput) {
				firstInput = input.Input
			}, outputs: []*output.InternalWrappedEvent{
				{OperatorsResult: &operators.Result{
					Matches:  map[string][]string{"tomcat": {}},
					Extracts: map[string][]string{},
				}},
			}}, Options: &protocols.ExecutorOptions{Progress: standardProgressBar}},
		}, Matchers: []*workflows.Matcher{{Name: stringslice.StringSlice{Value: "tomcat"}, Subtemplates: []*workflows.WorkflowTemplate{{Executers: []*workflows.ProtocolExecuterPair{{
			Executer: &mockExecuter{result: true, executeHook: func(input *contextargs.MetaInput) {
				secondInput = input.Input
			}}, Options: &protocols.ExecutorOptions{Progress: standardProgressBar}},
		}}}}}},
	}}

	engine := &Engine{}
	input := contextargs.NewWithInput("https://test.com")
	ctx := scan.NewScanContext(input)
	matched := engine.executeWorkflow(ctx, workflow)
	require.True(t, matched, "could not get correct match value")

	require.Equal(t, "https://test.com", firstInput, "could not get correct first input")
	require.Equal(t, "https://test.com", secondInput, "could not get correct second input")
}

func TestWorkflowsSubtemplatesWithMatcherNoMatch(t *testing.T) {
	progressBar, _ := progress.NewStatsTicker(0, false, false, false, 0)

	var firstInput, secondInput string
	workflow := &workflows.Workflow{Options: execOptions, Workflows: []*workflows.WorkflowTemplate{
		{Executers: []*workflows.ProtocolExecuterPair{{
			Executer: &mockExecuter{result: true, executeHook: func(input *contextargs.MetaInput) {
				firstInput = input.Input
			}, outputs: []*output.InternalWrappedEvent{
				{OperatorsResult: &operators.Result{
					Matches:  map[string][]string{"tomcat": {}},
					Extracts: map[string][]string{},
				}},
			}}, Options: &protocols.ExecutorOptions{Progress: progressBar}},
		}, Matchers: []*workflows.Matcher{{Name: stringslice.StringSlice{Value: "apache"}, Subtemplates: []*workflows.WorkflowTemplate{{Executers: []*workflows.ProtocolExecuterPair{{
			Executer: &mockExecuter{result: true, executeHook: func(input *contextargs.MetaInput) {
				secondInput = input.Input
			}}, Options: &protocols.ExecutorOptions{Progress: progressBar}},
		}}}}}},
	}}

	engine := &Engine{}
	input := contextargs.NewWithInput("https://test.com")
	ctx := scan.NewScanContext(input)
	matched := engine.executeWorkflow(ctx, workflow)
	require.False(t, matched, "could not get correct match value")

	require.Equal(t, "https://test.com", firstInput, "could not get correct first input")
	require.Equal(t, "", secondInput, "could not get correct second input")
}

type mockExecuter struct {
	result      bool
	executeHook func(input *contextargs.MetaInput)
	outputs     []*output.InternalWrappedEvent
}

// Compile compiles the execution generators preparing any requests possible.
func (m *mockExecuter) Compile() error {
	return nil
}

// Requests returns the total number of requests the rule will perform
func (m *mockExecuter) Requests() int {
	return 1
}

// Execute executes the protocol group and  returns true or false if results were found.
func (m *mockExecuter) Execute(ctx *scan.ScanContext) (bool, error) {
	if m.executeHook != nil {
		m.executeHook(ctx.Input.MetaInput)
	}
	return m.result, nil
}

// ExecuteWithResults executes the protocol requests and returns results instead of writing them.
func (m *mockExecuter) ExecuteWithResults(ctx *scan.ScanContext) ([]*output.ResultEvent, error) {
	if m.executeHook != nil {
		m.executeHook(ctx.Input.MetaInput)
	}
	for _, output := range m.outputs {
		ctx.LogEvent(output)
	}
	return ctx.GenerateResult(), nil
}
