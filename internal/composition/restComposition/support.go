package composition

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/gobuffalo/flect"
	"github.com/krateoplatformops/composition-dynamic-controller/internal/client/restclient"
	"github.com/krateoplatformops/composition-dynamic-controller/internal/text"
	"github.com/krateoplatformops/composition-dynamic-controller/internal/tools/apiaction"
	getter "github.com/krateoplatformops/composition-dynamic-controller/internal/tools/restclient"
	unstructuredtools "github.com/krateoplatformops/composition-dynamic-controller/internal/tools/unstructured"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

type RequestedParams struct {
	Parameters text.StringSet
	Query      text.StringSet
	Body       text.StringSet
}

type CallInfo struct {
	Path             string
	ReqParams        *RequestedParams
	IdentifierFields []string
	AltFields        map[string]string
}

type APIFuncDef func(ctx context.Context, cli *http.Client, path string, conf *restclient.RequestConfiguration) (*map[string]interface{}, error)

func APICallBuilder(cli *restclient.UnstructuredClient, info *getter.Info, action apiaction.APIAction) (apifunc APIFuncDef, callInfo *CallInfo, err error) {
	identifierFields := info.Resource.Identifiers
	for _, descr := range info.Resource.VerbsDescription {
		if strings.EqualFold(descr.Action, action.String()) {
			method, err := restclient.StringToApiCallType(descr.Method)
			if action == apiaction.FindBy {
				method = restclient.APICallsTypeFindBy
			}
			if err != nil {
				return nil, nil, fmt.Errorf("error converting method to api call type: %s", err)
			}
			params, query, err := cli.RequestedParams(descr.Method, descr.Path)
			if err != nil {
				return nil, nil, fmt.Errorf("error retrieving requested params: %s", err)
			}
			var body text.StringSet
			if descr.Method == "POST" || descr.Method == "PUT" || descr.Method == "PATCH" {
				body, err = cli.RequestedBody(descr.Method, descr.Path)
				if err != nil {
					return nil, nil, fmt.Errorf("error retrieving requested body params: %s", err)
				}
				if body == nil {
					body = text.StringSet{}
				}
			}

			callInfo := &CallInfo{
				Path: descr.Path,
				ReqParams: &RequestedParams{
					Parameters: params,
					Query:      query,
					Body:       body,
				},
				AltFields:        descr.AltFieldMapping,
				IdentifierFields: identifierFields,
			}
			switch method {
			case restclient.APICallsTypeGet:
				return cli.Get, callInfo, nil
			case restclient.APICallsTypePost:
				return cli.Post, callInfo, nil
			case restclient.APICallsTypeList:
				return cli.List, callInfo, nil
			case restclient.APICallsTypeDelete:
				return cli.Delete, callInfo, nil
			case restclient.APICallsTypePatch:
				return cli.Patch, callInfo, nil
			case restclient.APICallsTypeFindBy:
				return cli.FindBy, callInfo, nil
			case restclient.APICallsTypePut:
				return cli.Put, callInfo, nil
			}
		}
	}
	return nil, nil, nil //fmt.Errorf("impossible to build api call for action %s", action.String())
}

func BuildCallConfig(callInfo *CallInfo, statusFields map[string]interface{}, specFields map[string]interface{}) *restclient.RequestConfiguration {
	reqConfiguration := &restclient.RequestConfiguration{}
	reqConfiguration.Parameters = make(map[string]string)
	reqConfiguration.Query = make(map[string]string)
	mapBody := make(map[string]interface{})

	processFields(callInfo, specFields, reqConfiguration, mapBody)
	processFields(callInfo, statusFields, reqConfiguration, mapBody)
	reqConfiguration.Body = mapBody
	return reqConfiguration
}

func processFields(callInfo *CallInfo, fields map[string]interface{}, reqConfiguration *restclient.RequestConfiguration, mapBody map[string]interface{}) {
	for field, value := range fields {
		field, value = processAltFields(callInfo, field, value)
		if field == "" {
			continue
		}
		if callInfo.ReqParams.Parameters.Contains(field) {
			stringVal := fmt.Sprintf("%v", value)
			if stringVal == "" && reqConfiguration.Parameters[field] != "" {
				continue
			}
			reqConfiguration.Parameters[field] = stringVal
		} else if callInfo.ReqParams.Query.Contains(field) {
			stringVal := fmt.Sprintf("%v", value)
			if stringVal == "" && reqConfiguration.Query[field] != "" {
				continue
			}
			reqConfiguration.Query[field] = stringVal
		} else if callInfo.ReqParams.Body.Contains(field) {
			mapBody[field] = value
		}
	}
}

// if there are alternative fields, we need to check if the field is in the alternative field mapping
func processAltFields(callInfo *CallInfo, field string, value interface{}) (string, interface{}) {
	val := value
	for new, old := range callInfo.AltFields {
		// fmt.Println("Check before processing: ", new, old)
		split := strings.Split(new, ".")
		for i, altf := range split {
			if strings.Contains(altf, "[]") {
				arrayVal, ok := val.([]interface{})
				if !ok {
					continue
				}
				strVal := ""
				for _, value := range arrayVal {
					fmt.Println("len: ", len(split), i)
					_, v := processAltFields(callInfo, split[i+1], value)
					strv, ok := v.(string)
					if !ok {
						continue
					}
					strVal += strv
					strVal += ","
					// fmt.Println("After recursive call: ", f, strVal)
				}
				strVal = strings.TrimSuffix(strVal, ",")
				val = strVal
			} else {
				mapval, ok := val.(map[string]interface{})
				if ok {
					nval, ok := mapval[altf]
					if ok {
						val = nval
					}
				}
			}
		}
		if !reflect.DeepEqual(val, value) {
			return old, val
		}
	}

	f, ok := callInfo.AltFields[field]
	if ok {
		field = f
	}
	return field, value
}

func resolveObjectFromReferenceInfo(ref getter.ReferenceInfo, mg *unstructured.Unstructured, dyClient dynamic.Interface) (*unstructured.Unstructured, error) {
	gvrForReference := schema.GroupVersionResource{
		Group:    ref.GroupVersionKind.Group,
		Version:  ref.GroupVersionKind.Version,
		Resource: strings.ToLower(flect.Pluralize(ref.GroupVersionKind.Kind)),
	}

	all, err := dyClient.Resource(gvrForReference).
		List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting reference resource - %w", err)
	}
	if len(all.Items) == 0 {
		return nil, fmt.Errorf("no reference found for resource %s - len is zero", gvrForReference.Resource)
	}

	fieldValue, ok, err := unstructured.NestedString(mg.Object, "spec", ref.Field)
	if !ok {
		return nil, fmt.Errorf("spec field %s not found in reference resource", ref.Field)
	}
	if err != nil {
		return nil, fmt.Errorf("error getting spec field %s from reference resource - %w", ref.Field, err)
	}

	for _, item := range all.Items {
		statusMap, ok, err := unstructured.NestedMap(item.Object, "status")
		if !ok {
			return nil, fmt.Errorf("status field not found in reference resource")
		}
		if err != nil {
			return nil, fmt.Errorf("error getting status field from reference resource - %w", err)
		}

		for _, v := range statusMap {
			strField, ok := v.(string)
			if ok {
				if strField == fieldValue {
					return &item, nil
				}
			}
		}

		specMap, ok, err := unstructured.NestedMap(item.Object, "spec")
		if !ok {
			return nil, fmt.Errorf("spec field not found in reference resource")
		}
		if err != nil {
			return nil, fmt.Errorf("error getting spec field from reference resource - %w", err)
		}
		for _, v := range specMap {
			strField, ok := v.(string)
			if ok {
				if strField == fieldValue {
					return &item, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no reference found for resource %s", gvrForReference.Resource)
}

func isCRUpdated(def getter.Resource, mg *unstructured.Unstructured, rm map[string]interface{}) (bool, error) {
	specs, err := unstructuredtools.GetFieldsFromUnstructured(mg, "spec")
	if err != nil {
		return false, fmt.Errorf("error getting spec fields: %w", err)
	}
	if len(def.CompareList) > 0 {
		for _, field := range def.CompareList {
			if _, ok := rm[field]; !ok {
				return false, fmt.Errorf("field %s not found in response", field)
			}
			if !reflect.DeepEqual(specs[field], rm[field]) {
				return false, nil
			}
		}
		return true, nil
	}

	return compareExisting(mg, rm)
}

// recursive function to compare the field existing in the response rm with the field in the mg
func compareExisting(mg *unstructured.Unstructured, rm map[string]interface{}) (bool, error) {
	for k, v := range rm {
		if v == nil {
			continue
		}
		if reflect.TypeOf(v).Kind() == reflect.Map {
			m, ok := v.(map[string]interface{})
			if !ok {
				return false, fmt.Errorf("error converting value to map[string]interface{}")
			}
			if reflect.DeepEqual(m, mg.Object[k]) {
				continue
			}
			return compareExisting(mg, m)
		}
		if _, ok := mg.Object[k]; !ok {
			continue
		}

		if reflect.DeepEqual(v, mg.Object[k]) {
			continue
		}
		return false, nil
	}
	return true, nil

}
