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

var TsDataCenter = &types.TraversalSpec{
	Type: DataCenter,
	Path: VmFolder,
	SelectSet: []types.BaseSelectionSpec{
		&types.SelectionSpec{
			Name: TraverseFolders,
		},
	},
}

var TsRootFolder = &types.TraversalSpec{
	SelectionSpec: types.SelectionSpec{
		Name: TraverseFolders,
	},
	Type: Folder,
	Path: ChildEntity,
	SelectSet: []types.BaseSelectionSpec{
		TsDataCenter,
	},
}

type Credentials struct {
	Host     string
	User     string
	Password string
}

type TimeoutError struct {
}

func (e TimeoutError) Error() string {
	return "Timeout"
}

type Provider struct {
	Credentials
	client     *govmomi.Client
	ctx        context.Context
}

func (p *Provider) List() error {
	p.ctx = context.TODO()
	err := p.connect()
	if err != nil {
		return err
	}
	defer p.client.Logout(context.Background())
	handler := func(updates []types.ObjectUpdate) {
		for _, update := range updates {
			fmt.Println(update)
		}
	}

	return p.GetUpdates(false, handler)
}

func (p *Provider) Watch(ctx context.Context) error {
	p.ctx = ctx
	err := p.connect()
	if err != nil {
		return err
	}
	defer p.client.Logout(context.Background())
	handler := func(updates []types.ObjectUpdate) {
		for _, update := range updates {
			fmt.Println(update)
		}
	}
	
	return p.GetUpdates(true, handler)
}

func (p *Provider) GetUpdates(block bool, handler func([]types.ObjectUpdate)) error {
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
				return TimeoutError{}
			}
			continue
		}
		req.Version = updateSet.Version
		for _, fs := range updateSet.FilterSet {
			handler(fs.ObjectSet)
		}
		if !block {
			if updateSet.Truncated == nil || !*updateSet.Truncated {
				break
			}
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

func (p *Provider) filter(pc *property.Collector) *property.WaitFilter {
	return &property.WaitFilter{
		CreateFilter: types.CreateFilter{
			This: pc.Reference(),
			Spec: types.PropertyFilterSpec{
				ObjectSet: []types.ObjectSpec{
					p.objectSpec(),
				},
				PropSet: p.propertySpec(),
			},
		},
		Options: &types.WaitOptions{
			MaxWaitSeconds: types.NewInt32(5),
		},
	}
}

func (p *Provider) objectSpec() types.ObjectSpec {
	return types.ObjectSpec{
		Obj: p.client.ServiceContent.RootFolder,
		SelectSet: []types.BaseSelectionSpec{
			TsRootFolder,
		},
	}
}

func (p *Provider) propertySpec() []types.PropertySpec {
	return []types.PropertySpec{
		{
			Type: VirtualMachine,
			PathSet: []string{
				"summary",
			},
		},
	}
}
