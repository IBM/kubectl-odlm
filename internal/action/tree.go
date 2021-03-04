package action

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/gosuri/uitable"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/duration"

	odlmv1alpha1 "github.com/IBM/operand-deployment-lifecycle-manager/api/v1alpha1"
)

const (
	firstElemPrefix = `├─`
	lastElemPrefix  = `└─`
	pipe            = `│ `
)

var (
	gray = color.New(color.FgHiBlack)
)

type OperandRequestTree struct {
	RegistryMap            map[types.NamespacedName][]string
	OperandRequestInstance *odlmv1alpha1.OperandRequest
	SubscriptionList       []types.NamespacedName
}

type Tree struct {
	Config *Configuration
	Ctx    context.Context
	Table  *uitable.Table
}

func NewTree(cfg *Configuration) *Tree {
	return &Tree{
		Config: cfg,
	}
}

func (t *Tree) TreeView(opreqName string) {
	opreqTree := &OperandRequestTree{}
	t.printOpreq(opreqName, t.Config.Namespace, opreqTree)
	t.treeViewInner(t.Table, opreqTree)
}

func (t *Tree) printOpreq(opreqName, opreqNamespace string, opreqTree *OperandRequestTree) {
	key := types.NamespacedName{
		Namespace: opreqNamespace,
		Name:      opreqName,
	}
	opreq := &odlmv1alpha1.OperandRequest{}
	if err := t.Config.Client.Get(t.Ctx, key, opreq); err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	opreqTree.OperandRequestInstance = opreq
	opreqTree.RegistryMap = make(map[types.NamespacedName][]string)
	for _, req := range opreq.Spec.Requests {
		var operatorList []string
		for _, opt := range req.Operands {
			operatorList = append(operatorList, opt.Name)
		}
		var registryNamespace string
		if req.RegistryNamespace == "" {
			registryNamespace = t.Config.Namespace
		} else {
			registryNamespace = req.RegistryNamespace
		}
		key = types.NamespacedName{Namespace: registryNamespace, Name: req.Registry}
		opreqTree.RegistryMap[key] = operatorList
	}
	for opreg, opNameList := range opreqTree.RegistryMap {
		opregInstance := &odlmv1alpha1.OperandRegistry{}
		if err := t.Config.Client.Get(t.Ctx, opreg, opregInstance); err != nil {
			fmt.Println("Error: ", err)
			continue
		}
		for _, opName := range opNameList {
			find, opt := checkoptFromRegistry(opName, opregInstance)
			if !find {
				continue
			}
			opreqTree.SubscriptionList = append(opreqTree.SubscriptionList, opt)
		}
	}
}
func checkoptFromRegistry(name string, opreg *odlmv1alpha1.OperandRegistry) (find bool, opt types.NamespacedName) {
	for _, opt := range opreg.Spec.Operators {
		if opt.Name == name {
			var ns string
			if opt.Scope == "cluster" {
				ns = "openshift-operators"
			} else {
				ns = opt.Namespace
			}
			return true, types.NamespacedName{Namespace: ns, Name: name}
		}
	}
	return
}

func (t *Tree) treeViewInner(tbl *uitable.Table, opreqTree *OperandRequestTree) {
	opreqCreationTimestamp := opreqTree.OperandRequestInstance.GetCreationTimestamp()
	opreqAge := duration.HumanDuration(time.Since(opreqCreationTimestamp.Time))
	tbl.AddRow(opreqTree.OperandRequestInstance.Namespace, fmt.Sprintf("%s%s/%s",
		gray.Sprint(printPrefix("")),
		"OperandRequest",
		color.New(color.Bold).Sprint(opreqTree.OperandRequestInstance.Name)), opreqTree.OperandRequestInstance.Status.Phase, opreqAge)
	for i, sub := range opreqTree.SubscriptionList {
		subInstance := &v1alpha1.Subscription{}
		if err := t.Config.Client.Get(t.Ctx, sub, subInstance); err != nil {
			continue
		}
		var subPrefix string
		switch i {
		case len(opreqTree.SubscriptionList) - 1:
			subPrefix = lastElemPrefix
		default:
			subPrefix = firstElemPrefix
		}
		subCreationTimestamp := subInstance.GetCreationTimestamp()
		subAge := duration.HumanDuration(time.Since(subCreationTimestamp.Time))
		tbl.AddRow(sub.Namespace, fmt.Sprintf("%s%s/%s",
			gray.Sprint(printPrefix(subPrefix)),
			"Subscription",
			color.New(color.Bold).Sprint(sub.Name)), subInstance.Status.State, subAge)
		if subInstance.Status.InstalledCSV == "" {
			continue
		}
		csvInstance := &v1alpha1.ClusterServiceVersion{}
		if err := t.Config.Client.Get(t.Ctx, types.NamespacedName{Namespace: sub.Namespace, Name: subInstance.Status.InstalledCSV}, csvInstance); err != nil {
			continue
		}
		csvPrefix := subPrefix + lastElemPrefix
		csvCreationTimestamp := csvInstance.GetCreationTimestamp()
		csvAge := duration.HumanDuration(time.Since(csvCreationTimestamp.Time))
		tbl.AddRow(sub.Namespace, fmt.Sprintf("%s%s/%s",
			gray.Sprint(printPrefix(csvPrefix)),
			"ClusterServiceVersion",
			color.New(color.Bold).Sprint(subInstance.Status.InstalledCSV)), csvInstance.Status.Phase, csvAge)
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
			crCreationTimestamp := unstruct.GetCreationTimestamp()
			var phase string
			if _, ok := unstruct.Object["status"]; ok {
				if unstruct.Object["status"].(map[string]interface{})["phase"] != nil {
					phase = unstruct.Object["status"].(map[string]interface{})["phase"].(string)
				}
			}
			crAge := duration.HumanDuration(time.Since(crCreationTimestamp.Time))
			tbl.AddRow(sub.Namespace, fmt.Sprintf("%s%s/%s",
				gray.Sprint(printPrefix(crPrefix)),
				unstruct.Object["kind"].(string),
				color.New(color.Bold).Sprint(name)), phase, crAge)
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
