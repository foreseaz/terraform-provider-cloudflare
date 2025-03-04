package sdkv2provider

import (
	"fmt"
	"time"

	"github.com/cloudflare/cloudflare-go"
	"github.com/cloudflare/terraform-provider-cloudflare/internal/consts"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/pkg/errors"
)

func resourceCloudflareAccessApplicationSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		consts.AccountIDSchemaKey: {
			Description:   consts.AccountIDSchemaDescription,
			Type:          schema.TypeString,
			Optional:      true,
			Computed:      true,
			ConflictsWith: []string{consts.ZoneIDSchemaKey},
		},
		consts.ZoneIDSchemaKey: {
			Description:   consts.ZoneIDSchemaDescription,
			Type:          schema.TypeString,
			Optional:      true,
			Computed:      true,
			ConflictsWith: []string{consts.AccountIDSchemaKey},
		},
		"aud": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Application Audience (AUD) Tag of the application.",
		},
		"name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Friendly name of the Access Application.",
		},
		"domain": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "The primary hostname and path that Access will secure. If the app is visible in the App Launcher dashboard, this is the domain that will be displayed.",
		},
		"self_hosted_domains": {
			Type:     schema.TypeSet,
			Optional: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			Description: "List of domains that access will secure. Only present for self_hosted, vnc, and ssh applications. Always includes the value set as `domain`",
		},
		"type": {
			Type:         schema.TypeString,
			Optional:     true,
			Default:      "self_hosted",
			ValidateFunc: validation.StringInSlice([]string{"app_launcher", "bookmark", "biso", "dash_sso", "saas", "self_hosted", "ssh", "vnc", "warp"}, false),
			Description:  fmt.Sprintf("The application type. %s", renderAvailableDocumentationValuesStringSlice([]string{"app_launcher", "bookmark", "biso", "dash_sso", "saas", "self_hosted", "ssh", "vnc", "warp"})),
		},
		"session_duration": {
			Type:     schema.TypeString,
			Optional: true,
			Default:  "24h",
			DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
				appType := d.Get("type").(string)
				// Suppress the diff if it's a bookmark app type. Bookmarks don't have a session duration
				// field which always creates a diff because of the default '24h' value.
				if appType == "bookmark" {
					return true
				}

				return oldValue == newValue
			},
			ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
				v := val.(string)
				_, err := time.ParseDuration(v)
				if err != nil {
					errs = append(errs, fmt.Errorf(`%q only supports "ns", "us" (or "µs"), "ms", "s", "m", or "h" as valid units`, key))
				}
				return
			},
			Description: "How often a user will be forced to re-authorise. Must be in the format `48h` or `2h45m`",
		},
		"cors_headers": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "CORS configuration for the Access Application. See below for reference structure.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"allowed_methods": {
						Type:     schema.TypeSet,
						Optional: true,
						Elem: &schema.Schema{
							Type: schema.TypeString,
						},
						Description: "List of methods to expose via CORS.",
					},
					"allowed_origins": {
						Type:     schema.TypeSet,
						Optional: true,
						Elem: &schema.Schema{
							Type: schema.TypeString,
						},
						Description: "List of origins permitted to make CORS requests.",
					},
					"allowed_headers": {
						Type:     schema.TypeSet,
						Optional: true,
						Elem: &schema.Schema{
							Type: schema.TypeString,
						},
						Description: "List of HTTP headers to expose via CORS.",
					},
					"allow_all_methods": {
						Type:        schema.TypeBool,
						Optional:    true,
						Description: "Value to determine whether all methods are exposed.",
					},
					"allow_all_origins": {
						Type:        schema.TypeBool,
						Optional:    true,
						Description: "Value to determine whether all origins are permitted to make CORS requests.",
					},
					"allow_all_headers": {
						Type:        schema.TypeBool,
						Optional:    true,
						Description: "Value to determine whether all HTTP headers are exposed.",
					},
					"allow_credentials": {
						Type:        schema.TypeBool,
						Optional:    true,
						Description: "Value to determine if credentials (cookies, authorization headers, or TLS client certificates) are included with requests.",
					},
					"max_age": {
						Type:         schema.TypeInt,
						Optional:     true,
						ValidateFunc: validation.IntBetween(-1, 86400),
						Description:  "The maximum time a preflight request will be cached.",
					},
				},
			},
		},
		"saas_app": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "SaaS configuration for the Access Application.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"sp_entity_id": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "A globally unique name for an identity or service provider.",
					},
					"consumer_service_url": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The service provider's endpoint that is responsible for receiving and parsing a SAML assertion.",
					},
					"name_id_format": {
						Type:         schema.TypeString,
						Optional:     true,
						Default:      "email",
						ValidateFunc: validation.StringInSlice([]string{"email", "id"}, false),
						Description:  "The format of the name identifier sent to the SaaS application.",
					},
					"custom_attribute": {
						Type:        schema.TypeList,
						Optional:    true,
						Description: "Custom attribute mapped from IDPs.",
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"name": {
									Type:        schema.TypeString,
									Optional:    true,
									Description: "The name of the attribute as provided to the SaaS app.",
								},
								"name_format": {
									Type:        schema.TypeString,
									Optional:    true,
									Description: "A globally unique name for an identity or service provider.",
								},
								"friendly_name": {
									Type:        schema.TypeString,
									Optional:    true,
									Description: "A friendly name for the attribute as provided to the SaaS app.",
								},
								"required": {
									Type:        schema.TypeBool,
									Optional:    true,
									Description: "True if the attribute must be always present.",
								},
								"source": {
									Type:     schema.TypeList,
									Required: true,
									MaxItems: 1,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"name": {
												Type:        schema.TypeString,
												Required:    true,
												Description: "The name of the attribute as provided by the IDP.",
											},
										},
									},
								},
							},
						},
					},
					"idp_entity_id": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "The unique identifier for the SaaS application.",
					},
					"public_key": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "The public certificate that will be used to verify identities.",
					},
					"sso_endpoint": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "The endpoint where the SaaS application will send login requests.",
					},
				},
			},
		},
		"auto_redirect_to_identity": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Option to skip identity provider selection if only one is configured in `allowed_idps`.",
		},
		"enable_binding_cookie": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Option to provide increased security against compromised authorization tokens and CSRF attacks by requiring an additional \"binding\" cookie on requests.",
		},
		"allowed_idps": {
			Type:     schema.TypeSet,
			Optional: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			Description: "The identity providers selected for the application.",
		},
		"custom_deny_message": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Option that returns a custom error message when a user is denied access to the application.",
		},
		"custom_deny_url": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Option that redirects to a custom URL when a user is denied access to the application via identity based rules.",
		},
		"custom_non_identity_deny_url": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Option that redirects to a custom URL when a user is denied access to the application via non identity rules.",
		},
		"http_only_cookie_attribute": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Option to add the `HttpOnly` cookie flag to access tokens.",
		},
		"same_site_cookie_attribute": {
			Type:         schema.TypeString,
			Optional:     true,
			ValidateFunc: validation.StringInSlice([]string{"none", "lax", "strict"}, false),
			Description:  fmt.Sprintf("Defines the same-site cookie setting for access tokens. %s", renderAvailableDocumentationValuesStringSlice(([]string{"none", "lax", "strict"}))),
		},
		"logo_url": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Image URL for the logo shown in the app launcher dashboard.",
		},
		"skip_interstitial": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Option to skip the authorization interstitial when using the CLI.",
		},
		"app_launcher_visible": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
			Description: "Option to show/hide applications in App Launcher.",
		},
		"service_auth_401_redirect": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Option to return a 401 status code in service authentication rules on failed requests.",
		},
		"custom_pages": {
			Type:     schema.TypeSet,
			Optional: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			Description: "The custom pages selected for the application.",
		},
		"tags": {
			Type:     schema.TypeSet,
			Optional: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			Description: "The itags associated with the application.",
		},
	}
}

func convertCORSSchemaToStruct(d *schema.ResourceData) (*cloudflare.AccessApplicationCorsHeaders, error) {
	CORSConfig := cloudflare.AccessApplicationCorsHeaders{}

	if _, ok := d.GetOk("cors_headers"); ok {
		if allowedMethods, ok := d.GetOk("cors_headers.0.allowed_methods"); ok {
			CORSConfig.AllowedMethods = expandInterfaceToStringList(allowedMethods.(*schema.Set).List())
		}

		if allowedHeaders, ok := d.GetOk("cors_headers.0.allowed_headers"); ok {
			CORSConfig.AllowedHeaders = expandInterfaceToStringList(allowedHeaders.(*schema.Set).List())
		}

		if allowedOrigins, ok := d.GetOk("cors_headers.0.allowed_origins"); ok {
			CORSConfig.AllowedOrigins = expandInterfaceToStringList(allowedOrigins.(*schema.Set).List())
		}

		CORSConfig.AllowAllMethods = d.Get("cors_headers.0.allow_all_methods").(bool)
		CORSConfig.AllowAllHeaders = d.Get("cors_headers.0.allow_all_headers").(bool)
		CORSConfig.AllowAllOrigins = d.Get("cors_headers.0.allow_all_origins").(bool)
		CORSConfig.AllowCredentials = d.Get("cors_headers.0.allow_credentials").(bool)
		CORSConfig.MaxAge = d.Get("cors_headers.0.max_age").(int)

		// Prevent misconfigurations of CORS when `Access-Control-Allow-Origin` is
		// a wildcard (aka all origins) and using credentials.
		// See https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS/Errors/CORSNotSupportingCredentials
		if CORSConfig.AllowCredentials {
			if contains(CORSConfig.AllowedOrigins, "*") || CORSConfig.AllowAllOrigins {
				return nil, errors.New("CORS credentials are not permitted when all origins are allowed")
			}
		}

		// Ensure that should someone forget to set allowed methods (either
		// individually or *), we throw an error to prevent getting into an
		// unrecoverable state.
		if CORSConfig.AllowAllOrigins || len(CORSConfig.AllowedOrigins) > 1 {
			if CORSConfig.AllowAllMethods == false && len(CORSConfig.AllowedMethods) == 0 {
				return nil, errors.New("must set allowed_methods or allow_all_methods")
			}
		}

		// Ensure that should someone forget to set allowed origins (either
		// individually or *), we throw an error to prevent getting into an
		// unrecoverable state.
		if CORSConfig.AllowAllMethods || len(CORSConfig.AllowedMethods) > 1 {
			if CORSConfig.AllowAllOrigins == false && len(CORSConfig.AllowedOrigins) == 0 {
				return nil, errors.New("must set allowed_origins or allow_all_origins")
			}
		}
	}

	return &CORSConfig, nil
}

func convertCORSStructToSchema(d *schema.ResourceData, headers *cloudflare.AccessApplicationCorsHeaders) []interface{} {
	if _, ok := d.GetOk("cors_headers"); !ok {
		return []interface{}{}
	}

	if headers == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"allow_all_methods": headers.AllowAllMethods,
		"allow_all_headers": headers.AllowAllHeaders,
		"allow_all_origins": headers.AllowAllOrigins,
		"allow_credentials": headers.AllowCredentials,
		"max_age":           headers.MaxAge,
	}

	m["allowed_methods"] = flattenStringList(headers.AllowedMethods)
	m["allowed_headers"] = flattenStringList(headers.AllowedHeaders)
	m["allowed_origins"] = flattenStringList(headers.AllowedOrigins)

	return []interface{}{m}
}

func convertSAMLAttributeSchemaToStruct(data map[string]interface{}) cloudflare.SAMLAttributeConfig {
	var cfg cloudflare.SAMLAttributeConfig
	cfg.Name, _ = data["name"].(string)
	cfg.NameFormat, _ = data["name_format"].(string)
	cfg.Required, _ = data["required"].(bool)
	cfg.FriendlyName, _ = data["friendly_name"].(string)
	sourcesSlice, _ := data["source"].([]interface{})
	if len(sourcesSlice) != 0 {
		sourceMap, ok := sourcesSlice[0].(map[string]interface{})
		if ok {
			cfg.Source.Name, _ = sourceMap["name"].(string)
		}
	}

	return cfg
}

func convertSaasSchemaToStruct(d *schema.ResourceData) *cloudflare.SaasApplication {
	SaasConfig := cloudflare.SaasApplication{}
	if _, ok := d.GetOk("saas_app"); ok {
		SaasConfig.SPEntityID = d.Get("saas_app.0.sp_entity_id").(string)
		SaasConfig.ConsumerServiceUrl = d.Get("saas_app.0.consumer_service_url").(string)
		SaasConfig.NameIDFormat = d.Get("saas_app.0.name_id_format").(string)

		customAttributes, _ := d.Get("saas_app.0.custom_attribute").([]interface{})
		for _, customAttributes := range customAttributes {
			attributeAsMap := customAttributes.(map[string]interface{})
			SaasConfig.CustomAttributes = append(SaasConfig.CustomAttributes, convertSAMLAttributeSchemaToStruct(attributeAsMap))
		}
	}
	return &SaasConfig
}

func convertSAMLAttributeStructToSchema(attr cloudflare.SAMLAttributeConfig) map[string]interface{} {
	m := make(map[string]interface{})
	if attr.Name != "" {
		m["name"] = attr.Name
	}
	if attr.NameFormat != "" {
		m["name_format"] = attr.NameFormat
	}
	if attr.Required {
		m["required"] = true
	}
	if attr.FriendlyName != "" {
		m["friendly_name"] = attr.FriendlyName
	}
	if attr.Source.Name != "" {
		m["source"] = []interface{}{map[string]interface{}{"name": attr.Source.Name}}
	}
	return m
}

func convertSaasStructToSchema(d *schema.ResourceData, app *cloudflare.SaasApplication) []interface{} {
	if app == nil {
		return []interface{}{}
	}
	m := map[string]interface{}{
		"sp_entity_id":         app.SPEntityID,
		"consumer_service_url": app.ConsumerServiceUrl,
		"name_id_format":       app.NameIDFormat,
		"idp_entity_id":        app.IDPEntityID,
		"public_key":           app.PublicKey,
		"sso_endpoint":         app.SSOEndpoint,
	}

	var customAttributes []interface{}
	for _, attr := range app.CustomAttributes {
		customAttributes = append(customAttributes, convertSAMLAttributeStructToSchema(attr))
	}
	if len(customAttributes) != 0 {
		m["custom_attribute"] = customAttributes
	}

	return []interface{}{m}
}
