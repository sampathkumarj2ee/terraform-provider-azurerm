package aadb2c

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-provider-azurerm/internal/services/aadb2c/sdk/2021-04-01-preview/tenants"

	"github.com/hashicorp/terraform-provider-azurerm/helpers/azure"
	"github.com/hashicorp/terraform-provider-azurerm/internal/sdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/aadb2c/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tags"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/validation"
)

type AadB2cDirectoryDataSourceModel struct {
	BillingType           string            `tfschema:"billing_type"`
	DataResidencyLocation string            `tfschema:"data_residency_location"`
	DomainName            string            `tfschema:"domain_name"`
	EffectiveStartDate    string            `tfschema:"effective_start_date"`
	ResourceGroup         string            `tfschema:"resource_group_name"`
	Sku                   string            `tfschema:"sku_name"`
	Tags                  map[string]string `tfschema:"tags"`
	TenantId              string            `tfschema:"tenant_id"`
}

type AadB2cDirectoryDataSource struct{}

var _ sdk.DataSource = AadB2cDirectoryDataSource{}

func (r AadB2cDirectoryDataSource) ResourceType() string {
	return "azurerm_aadb2c_directory"
}

func (r AadB2cDirectoryDataSource) ModelObject() interface{} {
	return &AadB2cDirectoryModel{}
}

func (r AadB2cDirectoryDataSource) IDValidationFunc() pluginsdk.SchemaValidateFunc {
	return validate.B2CDirectoryID
}

func (r AadB2cDirectoryDataSource) Arguments() map[string]*pluginsdk.Schema {
	return map[string]*pluginsdk.Schema{
		"domain_name": {
			Type:         pluginsdk.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
		},

		"resource_group_name": azure.SchemaResourceGroupNameForDataSource(),
	}
}

func (r AadB2cDirectoryDataSource) Attributes() map[string]*pluginsdk.Schema {
	return map[string]*pluginsdk.Schema{
		"billing_type": {
			Type:     pluginsdk.TypeString,
			Computed: true,
		},

		"data_residency_location": {
			Type:     pluginsdk.TypeString,
			Computed: true,
		},

		"effective_start_date": {
			Type:     pluginsdk.TypeString,
			Computed: true,
		},

		"tenant_id": {
			Type:     pluginsdk.TypeString,
			Computed: true,
		},

		"sku_name": {
			Type:     pluginsdk.TypeString,
			Computed: true,
		},

		"tags": tags.SchemaDataSource(),
	}
}

func (r AadB2cDirectoryDataSource) Read() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 5 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			client := metadata.Client.AadB2c.TenantsClient
			subscriptionId := metadata.Client.Account.SubscriptionId

			var state AadB2cDirectoryModel
			if err := metadata.Decode(&state); err != nil {
				return fmt.Errorf("decoding: %+v", err)
			}

			id := tenants.NewB2CDirectoryID(subscriptionId, state.ResourceGroup, state.DomainName)

			metadata.Logger.Infof("Reading %s", id)
			resp, err := client.Get(ctx, id)
			if err != nil {
				if resp.HttpResponse.StatusCode == http.StatusNotFound {
					return metadata.MarkAsGone(id)
				}
				return fmt.Errorf("retrieving %s: %v", id, err)
			}

			model := resp.Model
			if model == nil {
				return fmt.Errorf("retrieving %s: model was nil", id)
			}

			state.DomainName = id.DirectoryName
			state.ResourceGroup = id.ResourceGroup

			if model.Location != nil {
				state.DataResidencyLocation = string(*model.Location)
			}

			if model.Sku != nil {
				state.Sku = string(model.Sku.Name)
			}

			if model.Tags != nil {
				state.Tags = *model.Tags
			}

			if properties := model.Properties; properties != nil {
				if billingConfig := properties.BillingConfig; billingConfig != nil {
					if billingConfig.BillingType != nil {
						state.BillingType = string(*billingConfig.BillingType)
					}
					if billingConfig.EffectiveStartDateUtc != nil {
						state.EffectiveStartDate = *billingConfig.EffectiveStartDateUtc
					}
				}

				if properties.TenantId != nil {
					state.TenantId = *properties.TenantId
				}
			}

			return metadata.Encode(&state)
		},
	}
}
