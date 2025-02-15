package datafactory

import (
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/datafactory/mgmt/2018-06-01/datafactory"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/azure"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/tf"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/datafactory/parse"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/datafactory/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/validation"
	"github.com/hashicorp/terraform-provider-azurerm/internal/timeouts"
	"github.com/hashicorp/terraform-provider-azurerm/utils"
)

func resourceDataFactoryLinkedServiceWeb() *pluginsdk.Resource {
	return &pluginsdk.Resource{
		Create: resourceDataFactoryLinkedServiceWebCreateUpdate,
		Read:   resourceDataFactoryLinkedServiceWebRead,
		Update: resourceDataFactoryLinkedServiceWebCreateUpdate,
		Delete: resourceDataFactoryLinkedServiceWebDelete,

		Importer: pluginsdk.ImporterValidatingResourceIdThen(func(id string) error {
			_, err := parse.LinkedServiceID(id)
			return err
		}, importDataFactoryLinkedService(datafactory.TypeBasicLinkedServiceTypeWeb)),

		Timeouts: &pluginsdk.ResourceTimeout{
			Create: pluginsdk.DefaultTimeout(30 * time.Minute),
			Read:   pluginsdk.DefaultTimeout(5 * time.Minute),
			Update: pluginsdk.DefaultTimeout(30 * time.Minute),
			Delete: pluginsdk.DefaultTimeout(30 * time.Minute),
		},

		Schema: map[string]*pluginsdk.Schema{
			"name": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.LinkedServiceDatasetName,
			},

			// TODO remove in 3.0
			"data_factory_name": {
				Type:         pluginsdk.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validate.DataFactoryName(),
				Deprecated:   "`data_factory_name` is deprecated in favour of `data_factory_id` and will be removed in version 3.0 of the AzureRM provider",
				ExactlyOneOf: []string{"data_factory_id"},
			},

			"data_factory_id": {
				Type:         pluginsdk.TypeString,
				Optional:     true, // TODO set to Required in 3.0
				Computed:     true, // TODO remove in 3.0
				ForceNew:     true,
				ValidateFunc: validate.DataFactoryID,
				ExactlyOneOf: []string{"data_factory_name"},
			},

			// There's a bug in the Azure API where this is returned in lower-case
			// BUG: https://github.com/Azure/azure-rest-api-specs/issues/5788
			"resource_group_name": azure.SchemaResourceGroupNameDiffSuppress(),

			"authentication_type": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},

			"url": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},

			"username": {
				Type:         pluginsdk.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},

			"password": {
				Type:         pluginsdk.TypeString,
				Optional:     true,
				Sensitive:    true,
				ValidateFunc: validation.StringIsNotEmpty,
			},

			"description": {
				Type:         pluginsdk.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},

			"integration_runtime_name": {
				Type:         pluginsdk.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},

			"parameters": {
				Type:     pluginsdk.TypeMap,
				Optional: true,
				Elem: &pluginsdk.Schema{
					Type: pluginsdk.TypeString,
				},
			},

			"annotations": {
				Type:     pluginsdk.TypeList,
				Optional: true,
				Elem: &pluginsdk.Schema{
					Type: pluginsdk.TypeString,
				},
			},

			"additional_properties": {
				Type:     pluginsdk.TypeMap,
				Optional: true,
				Elem: &pluginsdk.Schema{
					Type: pluginsdk.TypeString,
				},
			},
		},
	}
}

func resourceDataFactoryLinkedServiceWebCreateUpdate(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).DataFactory.LinkedServiceClient
	subscriptionId := meta.(*clients.Client).DataFactory.LinkedServiceClient.SubscriptionID
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	// TODO 3.0: remove/simplify this after deprecation
	var err error
	var dataFactoryId *parse.DataFactoryId
	if v := d.Get("data_factory_name").(string); v != "" {
		newDataFactoryId := parse.NewDataFactoryID(subscriptionId, d.Get("resource_group_name").(string), d.Get("data_factory_name").(string))
		dataFactoryId = &newDataFactoryId
	}
	if v := d.Get("data_factory_id").(string); v != "" {
		dataFactoryId, err = parse.DataFactoryID(v)
		if err != nil {
			return err
		}
	}

	id := parse.NewLinkedServiceID(subscriptionId, dataFactoryId.ResourceGroup, dataFactoryId.FactoryName, d.Get("name").(string))

	if d.IsNewResource() {
		existing, err := client.Get(ctx, id.ResourceGroup, id.FactoryName, id.Name, "")
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("checking for presence of existing Data Factory Web Anonymous %s: %+v", id, err)
			}
		}

		if existing.ID != nil && *existing.ID != "" {
			return tf.ImportAsExistsError("azurerm_data_factory_linked_service_web", *existing.ID)
		}
	}

	description := d.Get("description").(string)

	webLinkedService := &datafactory.WebLinkedService{
		Description: &description,
		Type:        datafactory.TypeBasicLinkedServiceTypeWeb,
	}

	url := d.Get("url").(string)
	authenticationType := d.Get("authentication_type").(string)

	if authenticationType == "Anonymous" {
		anonAuthProperties := &datafactory.WebAnonymousAuthentication{
			AuthenticationType: datafactory.AuthenticationType(authenticationType),
			URL:                utils.String(url),
		}
		webLinkedService.TypeProperties = anonAuthProperties
	}

	if authenticationType == "Basic" {
		username := d.Get("username").(string)
		password := d.Get("password").(string)
		passwordSecureString := datafactory.SecureString{
			Value: &password,
			Type:  datafactory.TypeSecureString,
		}
		basicAuthProperties := &datafactory.WebBasicAuthentication{
			AuthenticationType: datafactory.AuthenticationType(authenticationType),
			URL:                utils.String(url),
			Username:           username,
			Password:           passwordSecureString,
		}
		webLinkedService.TypeProperties = basicAuthProperties
	}

	if v, ok := d.GetOk("parameters"); ok {
		webLinkedService.Parameters = expandDataFactoryParameters(v.(map[string]interface{}))
	}

	if v, ok := d.GetOk("integration_runtime_name"); ok {
		webLinkedService.ConnectVia = expandDataFactoryLinkedServiceIntegrationRuntime(v.(string))
	}

	if v, ok := d.GetOk("additional_properties"); ok {
		webLinkedService.AdditionalProperties = v.(map[string]interface{})
	}

	if v, ok := d.GetOk("annotations"); ok {
		annotations := v.([]interface{})
		webLinkedService.Annotations = &annotations
	}

	linkedService := datafactory.LinkedServiceResource{
		Properties: webLinkedService,
	}

	if _, err := client.CreateOrUpdate(ctx, id.ResourceGroup, id.FactoryName, id.Name, linkedService, ""); err != nil {
		return fmt.Errorf("creating/updating Data Factory Web Anonymous %s: %+v", id, err)
	}

	d.SetId(id.ID())

	return resourceDataFactoryLinkedServiceWebRead(d, meta)
}

func resourceDataFactoryLinkedServiceWebRead(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).DataFactory.LinkedServiceClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.LinkedServiceID(d.Id())
	if err != nil {
		return err
	}

	dataFactoryId := parse.NewDataFactoryID(id.SubscriptionId, id.ResourceGroup, id.FactoryName)

	resp, err := client.Get(ctx, id.ResourceGroup, id.FactoryName, id.Name, "")
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("retrieving Data Factory Web %s: %+v", *id, err)
	}

	d.Set("name", resp.Name)
	d.Set("resource_group_name", id.ResourceGroup)
	// TODO 3.0: remove
	d.Set("data_factory_name", id.FactoryName)
	d.Set("data_factory_id", dataFactoryId.ID())

	web, ok := resp.Properties.AsWebLinkedService()
	if !ok {
		return fmt.Errorf("classifying Data Factory Web %s: Expected: %q Received: %q", *id, datafactory.TypeBasicLinkedServiceTypeWeb, *resp.Type)
	}

	isWebPropertiesLoaded := false

	anonProps, isAnonymousAuth := web.TypeProperties.AsWebAnonymousAuthentication()
	if isAnonymousAuth {
		d.Set("authentication_type", anonProps.AuthenticationType)
		d.Set("url", anonProps.URL)
		isWebPropertiesLoaded = true
	}

	if !isWebPropertiesLoaded {
		basicProps, isBasicAuth := web.TypeProperties.AsWebBasicAuthentication()
		if isBasicAuth {
			d.Set("authentication_type", basicProps.AuthenticationType)
			d.Set("url", basicProps.URL)
			d.Set("username", basicProps.Username)
			// d.Set("password", basicProps.Password)
		}
	}

	d.Set("additional_properties", web.AdditionalProperties)
	d.Set("description", web.Description)

	annotations := flattenDataFactoryAnnotations(web.Annotations)
	if err := d.Set("annotations", annotations); err != nil {
		return fmt.Errorf("setting `annotations`: %+v", err)
	}

	parameters := flattenDataFactoryParameters(web.Parameters)
	if err := d.Set("parameters", parameters); err != nil {
		return fmt.Errorf("setting `parameters`: %+v", err)
	}

	if connectVia := web.ConnectVia; connectVia != nil {
		if connectVia.ReferenceName != nil {
			d.Set("integration_runtime_name", connectVia.ReferenceName)
		}
	}

	return nil
}

func resourceDataFactoryLinkedServiceWebDelete(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).DataFactory.LinkedServiceClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.LinkedServiceID(d.Id())
	if err != nil {
		return err
	}

	response, err := client.Delete(ctx, id.ResourceGroup, id.FactoryName, id.Name)
	if err != nil {
		if !utils.ResponseWasNotFound(response) {
			return fmt.Errorf("deleting Data Factory Web %s: %+v", *id, err)
		}
	}

	return nil
}
