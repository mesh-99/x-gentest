package main

import (
    "encoding/json"
    "fmt"
    "gopkg.in/yaml.v2"
    "io/ioutil"
    "os"
    "path/filepath"

    extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
    v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

// Label keys.
const (
    LabelKeyNamePrefixForComposed = "crossplane.io/composite-resource."
    LabelKeyClaimName             = "crossplane.io/composite-resource-claim."
    LabelKeyClaimNamespace       = "crossplane.io/composite-resource-claim-namespace."
)

// Label values.
const (
    LabelValueDefault = "default"
)

// GetPropFields returns the fields from a map of schema properties
func GetPropFields(props map[string]extv1.JSONSchemaProps) []string {
    propFields := make([]string, len(props))
    i := 0
    for k := range props {
        propFields[i] = k
        i++
    }
    return propFields
}

func ForCompositeResource(xrd *v1.CompositeResourceDefinition) (*extv1.CustomResourceDefinition, error) {
    crd := &extv1.CustomResourceDefinition{
        Spec: extv1.CustomResourceDefinitionSpec{
            Scope:    extv1.ClusterScoped,
            Group:    xrd.Spec.Group,
            Names:    xrd.Spec.Names,
            Versions: make([]extv1.CustomResourceDefinitionVersion, len(xrd.Spec.Versions)),
        },
    }

    crd.SetName(xrd.GetName())
    crd.SetLabels(xrd.GetLabels())

    crd.Spec.Names.Categories = append(crd.Spec.Names.Categories, CategoryComposite)

    if err := dirToCRD(crd, xrd); err != nil {
        return nil, err
    }

    return crd, nil
}

func dirToCRD(crd *extv1.CustomResourceDefinition, xrd *v1.CompositeResourceDefinition) error {
    if xrd.Spec.Directory == "" {
        return nil
    }

    files, err := ioutil.ReadDir(xrd.Spec.Directory)
    if err != nil {
        return fmt.Errorf("failed to read directory: %w", err)
    }

    for _, f := range files {
        if f.IsDir() {
            continue
        }

        fileName := filepath.Join(xrd.Spec.Directory, f.Name())
        xrdFile, err := ioutil.ReadFile(fileName)
        if err != nil {
            return fmt.Errorf("failed to read file %s: %w", fileName, err)
        }

        xrd := &v1beta1.CompositeResourceDefinition{}
        err = yaml.Unmarshal(xrdFile, xrd)
        if err != nil {
            return fmt.Errorf("failed to unmarshal XRD file %s: %w", fileName, err)
        }

        if err := convertCRD(crd, xrd); err != nil {
            return fmt.Errorf("failed to convert XRD file %s: %w", fileName, err)
        }
    }

    return nil
}

func convertCRD(crd *extv1.CustomResourceDefinition, xrd *v1beta1.CompositeResourceDefinition) error {
    if err := mergeCRDSpec(crd, xrd.Spec); err != nil {
        return fmt.Errorf("failed to merge CRD spec: %w", err)
    }

    if err := mergeCRDStatus(crd, xrd.Spec.Status); err != nil {
        return fmt.Errorf("failed to merge CRD status: %w", err)
    }

    return nil
}
