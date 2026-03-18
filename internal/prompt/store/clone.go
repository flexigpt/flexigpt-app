package store

import (
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/prompt/spec"
)

func cloneAllTemplates(
	src map[bundleitemutils.BundleID]map[bundleitemutils.ItemID]spec.PromptTemplate,
) map[bundleitemutils.BundleID]map[bundleitemutils.ItemID]spec.PromptTemplate {
	dst := make(
		map[bundleitemutils.BundleID]map[bundleitemutils.ItemID]spec.PromptTemplate,
		len(src),
	)
	for bid, inner := range src {
		sub := make(map[bundleitemutils.ItemID]spec.PromptTemplate, len(inner))
		for tid, tpl := range inner {
			sub[tid] = clonePromptTemplate(tpl)
		}
		dst[bid] = sub
	}
	return dst
}

func clonePromptTemplate(in spec.PromptTemplate) spec.PromptTemplate {
	out := in

	if in.Tags != nil {
		out.Tags = append([]string(nil), in.Tags...)
	}
	if in.Blocks != nil {
		out.Blocks = append([]spec.MessageBlock(nil), in.Blocks...)
	}
	if in.Variables != nil {
		out.Variables = make([]spec.PromptVariable, len(in.Variables))
		for i, v := range in.Variables {
			vv := v
			if v.EnumValues != nil {
				vv.EnumValues = append([]string(nil), v.EnumValues...)
			}
			if v.Default != nil {
				def := *v.Default
				vv.Default = &def
			}
			out.Variables[i] = vv
		}
	}

	return out
}
