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
	unstructedOpreq := &unstructured.Unstructured{}
	err := t.Config.Scheme.Convert(opreqTree.OperandRequestInstance, unstructedOpreq, nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	reqReason := extractStatus(*unstructedOpreq)
	tbl.AddRow(opreqTree.OperandRequestInstance.Namespace, fmt.Sprintf("%s%s/%s",
		gray.Sprint(printPrefix("")),
		"OperandRequest",
		color.New(color.Bold).Sprint(opreqTree.OperandRequestInstance.Name)), reqReason, opreqAge)
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
			color.New(color.Bold).Sprint(sub.Name)), "", subAge)
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
		csvReason := extractCSVStatus(*csvInstance)
		tbl.AddRow(sub.Namespace, fmt.Sprintf("%s%s/%s",
			gray.Sprint(printPrefix(csvPrefix)),
			"ClusterServiceVersion",
			color.New(color.Bold).Sprint(subInstance.Status.InstalledCSV)), csvReason, csvAge)
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

		var crList []unstructured.Unstructured
		// Merge OandConfig and ClusterServiceVersion alm-examples
		for _, crTemplate := range crTemplates {

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

			crList = append(crList, unstruct)
		}
		for i, cr := range crList {
			var crPrefix string
			switch i {
			case len(crList) - 1:
				crPrefix = csvPrefix + lastElemPrefix
			default:
				crPrefix = csvPrefix + firstElemPrefix
			}
			crCreationTimestamp := cr.GetCreationTimestamp()
			name := cr.Object["metadata"].(map[string]interface{})["name"].(string)
			crReason := extractStatus(cr)
			crAge := duration.HumanDuration(time.Since(crCreationTimestamp.Time))
			tbl.AddRow(sub.Namespace, fmt.Sprintf("%s%s/%s",
				gray.Sprint(printPrefix(crPrefix)),
				cr.Object["kind"].(string),
				color.New(color.Bold).Sprint(name)), crReason, crAge)
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

func extractStatus(obj unstructured.Unstructured) (condReason string) {
	statusF, ok := obj.Object["status"]
	if !ok {
		return ""
	}
	statusV, ok := statusF.(map[string]interface{})
	if !ok {
		return ""
	}
	conditionsF, ok := statusV["conditions"]
	if !ok {
		return ""
	}
	conditionsV, ok := conditionsF.([]interface{})
	if !ok {
		return ""
	}

	for _, cond := range conditionsV {
		condM, ok := cond.(map[string]interface{})
		if !ok {
			return ""
		}
		condReason, _ = condM["reason"].(string)
		condStatus, _ := condM["status"].(string)
		if len(condStatus) != 0 {
			condReason = condStatus + "/" + condReason
		}
	}
	return
}

func extractCSVStatus(csv v1alpha1.ClusterServiceVersion) (condReason string) {
	conditions := csv.Status.Conditions
	for _, cond := range conditions {
		condReason = string(cond.Phase) + "/" + string(cond.Reason)
	}
	return
}
