package action

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/gosuri/uitable"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

const (
	firstElemPrefix = `├─`
	lastElemPrefix  = `└─`
	pipe            = `│ `
)

var (
	gray  = color.New(color.FgHiBlack)
)

type OperandRequestTree struct {
	Config                 *Configuration
	RegistryMap            map[types.NamespacedName][]string
	OperandRequestInstance types.NamespacedName
	SubscriptionList       []types.NamespacedName
	Ctx                    context.Context
}

func NewOperandRequestTree(cfg *Configuration) *OperandRequestTree {
	return &OperandRequestTree{
		Config: cfg,
	}
}

func (t *OperandRequestTree) TreeView() {
	tbl := uitable.New()
	tbl.Separator = "  "
	tbl.AddRow("NAMESPACE", "NAME")
	t.treeViewInner(tbl)
	fmt.Fprintln(color.Output, tbl)
}

func (t *OperandRequestTree) treeViewInner(tbl *uitable.Table) {

	tbl.AddRow(t.OperandRequestInstance.Namespace, fmt.Sprintf("%s%s/%s",
		gray.Sprint(printPrefix("")),
		"OperandRequest",
		color.New(color.Bold).Sprint(t.OperandRequestInstance.Name)))
	for i, sub := range t.SubscriptionList {
		subInstance := &v1alpha1.Subscription{}
		if err := t.Config.Client.Get(t.Ctx, sub, subInstance); err != nil {
			continue
		}
		var subPrefix string
		switch i {
		case len(t.SubscriptionList) - 1:
			subPrefix = lastElemPrefix
		default:
			subPrefix = firstElemPrefix
		}
		tbl.AddRow(sub.Namespace, fmt.Sprintf("%s%s/%s",
			gray.Sprint(printPrefix(subPrefix)),
			"Subscription",
			color.New(color.Bold).Sprint(sub.Name)))
		if subInstance.Status.InstalledCSV == "" {
			continue
		}
		csvInstance := &v1alpha1.ClusterServiceVersion{}
		if err := t.Config.Client.Get(t.Ctx, types.NamespacedName{Namespace: sub.Namespace, Name: subInstance.Status.InstalledCSV}, csvInstance); err != nil {
			continue
		}
		csvPrefix := subPrefix + lastElemPrefix
		tbl.AddRow(sub.Namespace, fmt.Sprintf("%s%s/%s",
			gray.Sprint(printPrefix(csvPrefix)),
			"ClusterServiceVersion",
			color.New(color.Bold).Sprint(subInstance.Status.InstalledCSV)))
		almExamples, ok := csvInstance.Annotations["alm-examples"]
		if !ok {
			continue
		}
		// Create a slice for crTemplates
		var crTemplates []interface{}

		// Convert CR template string to slice
		err := json.Unmarshal([]byte(almExamples), &crTemplates)
		if err != nil {
			continue
		}

		// Merge OperandConfig and ClusterServiceVersion alm-examples
		for i, crTemplate := range crTemplates {

			// Create an unstructed object for CR and request its value to CR template
			var unstruct unstructured.Unstructured
			unstruct.Object = crTemplate.(map[string]interface{})

			name := unstruct.Object["metadata"].(map[string]interface{})["name"].(string)

			err := t.Config.Client.Get(t.Ctx, types.NamespacedName{
				Name:      name,
				Namespace: sub.Namespace,
			}, &unstruct)

			if err != nil {
				continue
			}

			var crPrefix string
			switch i {
			case len(crTemplates) - 1:
				crPrefix = csvPrefix + lastElemPrefix
			default:
				crPrefix = csvPrefix + firstElemPrefix
			}
			tbl.AddRow(sub.Namespace, fmt.Sprintf("%s%s/%s",
				gray.Sprint(printPrefix(crPrefix)),
				unstruct.Object["kind"].(string),
				color.New(color.Bold).Sprint(name)))
		}
	}

}

func printPrefix(p string) string {
	if strings.HasSuffix(p, firstElemPrefix) {
		p = strings.Replace(p, firstElemPrefix, pipe, strings.Count(p, firstElemPrefix)-1)
	} else {
		p = strings.ReplaceAll(p, firstElemPrefix, pipe)
	}

	if strings.HasSuffix(p, lastElemPrefix) {
		p = strings.Replace(p, lastElemPrefix, strings.Repeat(" ", len([]rune(lastElemPrefix))), strings.Count(p, lastElemPrefix)-1)
	} else {
		p = strings.ReplaceAll(p, lastElemPrefix, strings.Repeat(" ", len([]rune(lastElemPrefix))))
	}
	return p
}
