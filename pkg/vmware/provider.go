package vmware

import (
	"context"
	"fmt"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/types"
	liburl "net/url"
)

const (
	VirtualMachine = "VirtualMachine"
	DataCenter     = "Datacenter"
)

const (
	Folder          = "Folder"
	VmFolder        = "vmFolder"
	ChildEntity     = "childEntity"
	TraverseFolders = "traverseFolders"
)

type Credentials struct {
	Host     string
	User     string
	Password string
}

type Provider struct {
	Credentials
	Reconciled bool
	client     *govmomi.Client
	ctx        context.Context
}

func (p *Provider) Start(ctx context.Context) error {
	p.ctx = ctx
	err := p.connect()
	if err != nil {
		return err
	}
	defer p.client.Logout(context.Background())
	p.Watch()

	return nil
}

func (p *Provider) Watch() error {
	pc := property.DefaultCollector(p.client.Client)
	pc, err := pc.Create(p.ctx)
	if err != nil {
		return err
	}
	defer pc.Destroy(context.Background())
	filter := p.filter(pc)
	err = pc.CreateFilter(p.ctx, filter.CreateFilter)
	if err != nil {
		return err
	}
	req := types.WaitForUpdatesEx{
		This:    pc.Reference(),
		Options: filter.Options,
	}
	for {
		res, err := methods.WaitForUpdatesEx(p.ctx, p.client, &req)
		if err != nil {
			if p.ctx.Err() == context.Canceled {
				pc.CancelWaitForUpdates(context.Background())
				break
			}
			return err
		}
		updateSet := res.Returnval
		if updateSet == nil {
			if req.Options != nil && req.Options.MaxWaitSeconds != nil {
				return nil
			}
			continue
		}
		if updateSet.Truncated == nil || !*updateSet.Truncated {
			p.Reconciled = true
		}
		req.Version = updateSet.Version
		for _, fs := range updateSet.FilterSet {
			p.updated(fs.ObjectSet)
		}
	}

	return nil
}

func (p *Provider) connect() error {
	insecure := true
	url := &liburl.URL{
		Scheme: "https",
		User:   liburl.UserPassword(p.User, p.Password),
		Host:   p.Host,
		Path:   vim25.Path,
	}
	client, err := govmomi.NewClient(p.ctx, url, insecure)
	if err != nil {
		return err
	}

	p.client = client

	return nil
}

func (p *Provider) updated(updates []types.ObjectUpdate) error {
	for _, update := range updates {
		fmt.Println(update)
	}

	return nil
}

func (p *Provider) filter(pc *property.Collector) *property.WaitFilter {
	return &property.WaitFilter{
		CreateFilter: types.CreateFilter{
			This: pc.Reference(),
			Spec: types.PropertyFilterSpec{
				ObjectSet: []types.ObjectSpec{
					p.objectSpecification(),
				},
				PropSet: p.propertySpecification(),
			},
		},
		Options: &types.WaitOptions{
			MaxWaitSeconds: types.NewInt32(5),
		},
	}
}

func (p *Provider) objectSpecification() types.ObjectSpec {
	return types.ObjectSpec{
		Obj: p.client.ServiceContent.RootFolder,
		SelectSet: []types.BaseSelectionSpec{
			&types.TraversalSpec{
				SelectionSpec: types.SelectionSpec{
					Name: TraverseFolders,
				},
				Type: Folder,
				Path: ChildEntity,
				SelectSet: []types.BaseSelectionSpec{
					&types.TraversalSpec{
						Type: DataCenter,
						Path: VmFolder,
						SelectSet: []types.BaseSelectionSpec{
							&types.SelectionSpec{
								Name: TraverseFolders,
							},
						},
					},
				},
			},
		},
	}
}

func (p *Provider) propertySpecification() []types.PropertySpec {
	return []types.PropertySpec{
		{
			Type: VirtualMachine,
			PathSet: []string{
				"summary",
			},
		},
	}
}
