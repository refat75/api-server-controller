// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1 "github.com/refat75/apiServerController/pkg/apis/appscode.refat.dev/v1"
	appscoderefatdevv1 "github.com/refat75/apiServerController/pkg/generated/applyconfiguration/appscode.refat.dev/v1"
	typedappscoderefatdevv1 "github.com/refat75/apiServerController/pkg/generated/clientset/versioned/typed/appscode.refat.dev/v1"
	gentype "k8s.io/client-go/gentype"
)

// fakeApiservers implements ApiserverInterface
type fakeApiservers struct {
	*gentype.FakeClientWithListAndApply[*v1.Apiserver, *v1.ApiserverList, *appscoderefatdevv1.ApiserverApplyConfiguration]
	Fake *FakeAppscodeV1
}

func newFakeApiservers(fake *FakeAppscodeV1, namespace string) typedappscoderefatdevv1.ApiserverInterface {
	return &fakeApiservers{
		gentype.NewFakeClientWithListAndApply[*v1.Apiserver, *v1.ApiserverList, *appscoderefatdevv1.ApiserverApplyConfiguration](
			fake.Fake,
			namespace,
			v1.SchemeGroupVersion.WithResource("apiservers"),
			v1.SchemeGroupVersion.WithKind("Apiserver"),
			func() *v1.Apiserver { return &v1.Apiserver{} },
			func() *v1.ApiserverList { return &v1.ApiserverList{} },
			func(dst, src *v1.ApiserverList) { dst.ListMeta = src.ListMeta },
			func(list *v1.ApiserverList) []*v1.Apiserver { return gentype.ToPointerSlice(list.Items) },
			func(list *v1.ApiserverList, items []*v1.Apiserver) { list.Items = gentype.FromPointerSlice(items) },
		),
		fake,
	}
}
