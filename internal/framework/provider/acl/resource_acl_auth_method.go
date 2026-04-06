// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package acl

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-nomad/nomad"
)

var _ resource.Resource = &ACLAuthMethodResource{}
var _ resource.ResourceWithConfigure = &ACLAuthMethodResource{}
var _ resource.ResourceWithImportState = &ACLAuthMethodResource{}
var _ resource.ResourceWithModifyPlan = &ACLAuthMethodResource{}
var _ resource.ResourceWithUpgradeState = &ACLAuthMethodResource{}

type ACLAuthMethodResource struct {
	providerConfig nomad.ProviderConfig
}

func NewACLAuthMethodResource() resource.Resource {
	return &ACLAuthMethodResource{}
}

type aclAuthMethodModel struct {
	ID              types.String              `tfsdk:"id"`
	Name            types.String              `tfsdk:"name"`
	Type            types.String              `tfsdk:"type"`
	TokenLocality   types.String              `tfsdk:"token_locality"`
	MaxTokenTTL     types.String              `tfsdk:"max_token_ttl"`
	TokenNameFormat types.String              `tfsdk:"token_name_format"`
	Default         types.Bool                `tfsdk:"default"`
	Config          *aclAuthMethodConfigModel `tfsdk:"config"`
}

type aclAuthMethodConfigModel struct {
	JWTValidationPubKeys  types.List                         `tfsdk:"jwt_validation_pub_keys"`
	JWKSURL               types.String                       `tfsdk:"jwks_url"`
	JWKSCACert            types.String                       `tfsdk:"jwks_ca_cert"`
	OIDCDiscoveryURL      types.String                       `tfsdk:"oidc_discovery_url"`
	OIDCClientID          types.String                       `tfsdk:"oidc_client_id"`
	OIDCClientSecret      types.String                       `tfsdk:"oidc_client_secret"`
	OIDCClientSecretWO    types.String                       `tfsdk:"oidc_client_secret_wo"`
	OIDCClientSecretWOVer types.Int64                        `tfsdk:"oidc_client_secret_wo_version"`
	OIDCClientAssertion   *aclAuthMethodClientAssertionModel `tfsdk:"oidc_client_assertion"`
	OIDCEnablePKCE        types.Bool                         `tfsdk:"oidc_enable_pkce"`
	OIDCDisableUserInfo   types.Bool                         `tfsdk:"oidc_disable_userinfo"`
	OIDCScopes            types.List                         `tfsdk:"oidc_scopes"`
	BoundAudiences        types.List                         `tfsdk:"bound_audiences"`
	BoundIssuer           types.List                         `tfsdk:"bound_issuer"`
	AllowedRedirectURIs   types.List                         `tfsdk:"allowed_redirect_uris"`
	DiscoveryCAPEM        types.List                         `tfsdk:"discovery_ca_pem"`
	SigningAlgs           types.List                         `tfsdk:"signing_algs"`
	ExpirationLeeway      types.String                       `tfsdk:"expiration_leeway"`
	NotBeforeLeeway       types.String                       `tfsdk:"not_before_leeway"`
	ClockSkewLeeway       types.String                       `tfsdk:"clock_skew_leeway"`
	ClaimMappings         types.Map                          `tfsdk:"claim_mappings"`
	ListClaimMappings     types.Map                          `tfsdk:"list_claim_mappings"`
	VerboseLogging        types.Bool                         `tfsdk:"verbose_logging"`
}

type aclAuthMethodClientAssertionModel struct {
	Audience     types.List                    `tfsdk:"audience"`
	ExtraHeaders types.Map                     `tfsdk:"extra_headers"`
	KeyAlgorithm types.String                  `tfsdk:"key_algorithm"`
	KeySource    types.String                  `tfsdk:"key_source"`
	PrivateKey   *aclAuthMethodPrivateKeyModel `tfsdk:"private_key"`
}

type aclAuthMethodPrivateKeyModel struct {
	PemKey      types.String `tfsdk:"pem_key"`
	PemKeyWO    types.String `tfsdk:"pem_key_wo"`
	PemKeyWOVer types.Int64  `tfsdk:"pem_key_wo_version"`
	PemKeyFile  types.String `tfsdk:"pem_key_file"`
	PemCert     types.String `tfsdk:"pem_cert"`
	PemCertFile types.String `tfsdk:"pem_cert_file"`
	KeyIDHeader types.String `tfsdk:"key_id_header"`
	KeyID       types.String `tfsdk:"key_id"`
}

func (r *ACLAuthMethodResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_acl_auth_method"
}

func (r *ACLAuthMethodResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Manages an ACL Auth Method in Nomad.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The identifier of the ACL Auth Method (same as name).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The identifier of the ACL Auth Method.",
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: `ACL Auth Method SSO workflow type. Currently, the only supported types are "OIDC" and "JWT".`,
			},
			"token_locality": schema.StringAttribute{
				Required:    true,
				Description: `Defines whether the ACL Auth Method creates a local or global token when performing SSO login. This field must be set to either "local" or "global".`,
			},
			"max_token_ttl": schema.StringAttribute{
				Required:    true,
				Description: "Defines the maximum life of a token created by this method.",
				PlanModifiers: []planmodifier.String{
					maxTokenTTLPlanModifier{},
				},
			},
			"token_name_format": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("${auth_method_type}-${auth_method_name}"),
				Description: "Defines the token format for the authenticated users. This can be lightly templated using HIL '${foo}' syntax.",
			},
			"default": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Defines whether this ACL Auth Method is to be set as default.",
			},
		},
		Blocks: map[string]schema.Block{
			"config": schema.SingleNestedBlock{
				Description: "Configuration specific to the auth method provider.",
				Attributes: map[string]schema.Attribute{
					"jwt_validation_pub_keys": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Description: "List of PEM-encoded public keys to use to authenticate signatures locally.",
					},
					"jwks_url": schema.StringAttribute{
						Optional:    true,
						Description: "JSON Web Key Sets url for authenticating signatures.",
					},
					"jwks_ca_cert": schema.StringAttribute{
						Optional:    true,
						Description: "PEM encoded CA cert for use by the TLS client used to talk with the JWKS server.",
					},
					"oidc_discovery_url": schema.StringAttribute{
						Optional:    true,
						Description: "The OIDC Discovery URL, without any .well-known component (base path).",
					},
					"oidc_client_id": schema.StringAttribute{
						Optional:    true,
						Description: "The OAuth Client ID configured with the OIDC provider.",
					},
					"oidc_client_secret": schema.StringAttribute{
						Optional:           true,
						Sensitive:          true,
						Description:        "The OAuth Client Secret configured with the OIDC provider. Deprecated: use oidc_client_secret_wo instead.",
						DeprecationMessage: "Use oidc_client_secret_wo to avoid storing the secret in Terraform state.",
						Validators: []validator.String{
							stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("oidc_client_secret_wo")),
						},
					},
					"oidc_client_secret_wo": schema.StringAttribute{
						Optional:    true,
						WriteOnly:   true,
						Description: "The OAuth Client Secret configured with the OIDC provider. This value is write-only and will not be stored in Terraform state.",
						Validators: []validator.String{
							stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("oidc_client_secret")),
						},
					},
					"oidc_client_secret_wo_version": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Description: "Version counter for oidc_client_secret_wo. Increments automatically when the write-only secret changes, or set manually to trigger an update.",
					},
					"oidc_enable_pkce": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						Description: "Nomad include PKCE challenge in OIDC auth requests.",
					},
					"oidc_disable_userinfo": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						Description: "Nomad will not make a request to the identity provider to get OIDC UserInfo.",
					},
					"oidc_scopes": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Description: "List of OIDC scopes.",
					},
					"bound_audiences": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Description: "List of auth claims that are valid for login.",
					},
					"bound_issuer": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Description: "The value against which to match the iss claim in a JWT.",
					},
					"allowed_redirect_uris": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Description: "A list of allowed values that can be used for the redirect URI.",
					},
					"discovery_ca_pem": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Description: "PEM encoded CA certs for use by the TLS client used to talk with the OIDC Discovery URL.",
					},
					"signing_algs": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Description: "A list of supported signing algorithms.",
					},
					"expiration_leeway": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("0s"),
						Description: `Duration of leeway when validating expiration of a JWT in the form of a time duration such as "5m" or "1h".`,
					},
					"not_before_leeway": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("0s"),
						Description: `Duration of leeway when validating not before values of a token in the form of a time duration such as "5m" or "1h".`,
					},
					"clock_skew_leeway": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("0s"),
						Description: `Duration of leeway when validating all claims in the form of a time duration such as "5m" or "1h".`,
					},
					"claim_mappings": schema.MapAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Description: "Mappings of claims (key) that will be copied to a metadata field (value).",
					},
					"list_claim_mappings": schema.MapAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Description: "Mappings of list claims (key) that will be copied to a metadata field (value).",
					},
					"verbose_logging": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						Description: "Enable OIDC verbose logging on the Nomad server.",
					},
				},
				Blocks: map[string]schema.Block{
					"oidc_client_assertion": schema.SingleNestedBlock{
						Description: "Configuration for OIDC client assertion / private key JWT.",
						Attributes: map[string]schema.Attribute{
							"audience": schema.ListAttribute{
								ElementType: types.StringType,
								Optional:    true,
								Computed:    true,
								Description: "List of audiences to accept the JWT.",
							},
							"extra_headers": schema.MapAttribute{
								ElementType: types.StringType,
								Optional:    true,
								Description: "Additional headers to include on the JWT.",
							},
							"key_algorithm": schema.StringAttribute{
								Optional:    true,
								Computed:    true,
								Description: "Algorithm of the key used to sign the JWT.",
							},
							"key_source": schema.StringAttribute{
								Optional:    true,
								Description: "The source of the key Nomad will use to sign the JWT.",
							},
						},
						Blocks: map[string]schema.Block{
							"private_key": schema.SingleNestedBlock{
								Description: "Configuration for a custom private key to sign the JWT.",
								Attributes: map[string]schema.Attribute{
									"pem_key": schema.StringAttribute{
										Optional:           true,
										Sensitive:          true,
										Description:        "RSA private key PEM to use to sign the JWT. Deprecated: use pem_key_wo instead.",
										DeprecationMessage: "Use pem_key_wo to avoid storing the private key in Terraform state.",
										Validators: []validator.String{
											stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("pem_key_wo")),
										},
									},
									"pem_key_wo": schema.StringAttribute{
										Optional:    true,
										WriteOnly:   true,
										Description: "RSA private key PEM to use to sign the JWT. This value is write-only and will not be stored in Terraform state.",
										Validators: []validator.String{
											stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("pem_key")),
										},
									},
									"pem_key_wo_version": schema.Int64Attribute{
										Optional:    true,
										Computed:    true,
										Description: "Version counter for pem_key_wo. Increments automatically when the write-only key changes, or set manually to trigger an update.",
									},
									"pem_key_file": schema.StringAttribute{
										Optional:    true,
										Description: "Path to an RSA private key PEM on Nomad servers to use to sign the JWT.",
									},
									"pem_cert": schema.StringAttribute{
										Optional:    true,
										Description: "An x509 certificate PEM to derive a key ID header.",
									},
									"pem_cert_file": schema.StringAttribute{
										Optional:    true,
										Description: "Path to an x509 certificate PEM on Nomad servers to derive a key ID header.",
									},
									"key_id_header": schema.StringAttribute{
										Optional:    true,
										Computed:    true,
										Default:     stringdefault.StaticString("x5t#S256"),
										Description: "Name of the header the IDP will use to find the cert to verify the JWT signature.",
									},
									"key_id": schema.StringAttribute{
										Optional:    true,
										Description: "Specific 'kid' header to set on the JWT.",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *ACLAuthMethodResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	metaFunc, ok := req.ProviderData.(func() any)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected func() any, got %T.", req.ProviderData),
		)
		return
	}

	providerConfig, ok := metaFunc().(nomad.ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Meta Type",
			fmt.Sprintf("Expected nomad.ProviderConfig, got %T.", metaFunc()),
		)
		return
	}

	r.providerConfig = providerConfig
}

func (r *ACLAuthMethodResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data aclAuthMethodModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var configData aclAuthMethodModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &configData)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if data.Config != nil && configData.Config != nil {
		data.Config.OIDCClientSecretWO = configData.Config.OIDCClientSecretWO
		if data.Config.OIDCClientAssertion != nil && configData.Config.OIDCClientAssertion != nil &&
			data.Config.OIDCClientAssertion.PrivateKey != nil && configData.Config.OIDCClientAssertion.PrivateKey != nil {
			data.Config.OIDCClientAssertion.PrivateKey.PemKeyWO = configData.Config.OIDCClientAssertion.PrivateKey.PemKeyWO
		}
	}

	authMethod, err := modelToAPI(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error building ACL auth method", err.Error())
		return
	}

	tflog.Debug(ctx, "Creating ACL auth method", map[string]interface{}{"name": authMethod.Name})
	result, _, err := r.providerConfig.Client().ACLAuthMethods().Create(authMethod, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error creating ACL auth method", err.Error())
		return
	}
	tflog.Debug(ctx, "Created ACL auth method", map[string]interface{}{"name": result.Name})

	if data.Config != nil {
		if !data.Config.OIDCClientSecretWO.IsNull() {
			woHash := calculateWOHash(data.Config.OIDCClientSecretWO.ValueString())
			resp.Diagnostics.Append(setPrivateWOHash(ctx, resp.Private, "oidc_client_secret_wo_hash", woHash)...)
			if resp.Diagnostics.HasError() {
				return
			}
			if data.Config.OIDCClientSecretWOVer.IsNull() || data.Config.OIDCClientSecretWOVer.IsUnknown() {
				data.Config.OIDCClientSecretWOVer = types.Int64Value(1)
			}
		} else {
			data.Config.OIDCClientSecretWOVer = types.Int64Null()
		}

		if data.Config.OIDCClientAssertion != nil && data.Config.OIDCClientAssertion.PrivateKey != nil {
			if !data.Config.OIDCClientAssertion.PrivateKey.PemKeyWO.IsNull() {
				pkHash := calculateWOHash(data.Config.OIDCClientAssertion.PrivateKey.PemKeyWO.ValueString())
				resp.Diagnostics.Append(setPrivateWOHash(ctx, resp.Private, "pem_key_wo_hash", pkHash)...)
				if resp.Diagnostics.HasError() {
					return
				}
				if data.Config.OIDCClientAssertion.PrivateKey.PemKeyWOVer.IsNull() || data.Config.OIDCClientAssertion.PrivateKey.PemKeyWOVer.IsUnknown() {
					data.Config.OIDCClientAssertion.PrivateKey.PemKeyWOVer = types.Int64Value(1)
				}
			} else {
				data.Config.OIDCClientAssertion.PrivateKey.PemKeyWOVer = types.Int64Null()
			}
		}
	}

	fetched, _, err := r.providerConfig.Client().ACLAuthMethods().Get(result.Name, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error reading ACL auth method after create", err.Error())
		return
	}

	apiToModel(ctx, fetched, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ACLAuthMethodResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data aclAuthMethodModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	tflog.Debug(ctx, "Reading ACL auth method", map[string]interface{}{"name": name})
	fetched, _, err := r.providerConfig.Client().ACLAuthMethods().Get(name, nil)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading ACL auth method", err.Error())
		return
	}
	tflog.Debug(ctx, "Read ACL auth method", map[string]interface{}{"name": name})

	var priorClientSecret, priorPemKey string
	if data.Config != nil {
		priorClientSecret = data.Config.OIDCClientSecret.ValueString()
		if data.Config.OIDCClientAssertion != nil && data.Config.OIDCClientAssertion.PrivateKey != nil {
			priorPemKey = data.Config.OIDCClientAssertion.PrivateKey.PemKey.ValueString()
		}
	}

	apiToModel(ctx, fetched, &data)

	if data.Config != nil {
		if priorClientSecret != "" {
			data.Config.OIDCClientSecret = types.StringValue(priorClientSecret)
		}
		if data.Config.OIDCClientAssertion != nil && data.Config.OIDCClientAssertion.PrivateKey != nil && priorPemKey != "" {
			data.Config.OIDCClientAssertion.PrivateKey.PemKey = types.StringValue(priorPemKey)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ACLAuthMethodResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan aclAuthMethodModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var configData aclAuthMethodModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &configData)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.Config != nil && configData.Config != nil {
		plan.Config.OIDCClientSecretWO = configData.Config.OIDCClientSecretWO
		if plan.Config.OIDCClientAssertion != nil && configData.Config.OIDCClientAssertion != nil &&
			plan.Config.OIDCClientAssertion.PrivateKey != nil && configData.Config.OIDCClientAssertion.PrivateKey != nil {
			plan.Config.OIDCClientAssertion.PrivateKey.PemKeyWO = configData.Config.OIDCClientAssertion.PrivateKey.PemKeyWO
		}
	}

	if plan.Config != nil && configData.Config != nil {
		if !configData.Config.OIDCClientSecretWO.IsNull() {
			newSecretHash := calculateWOHash(configData.Config.OIDCClientSecretWO.ValueString())
			resp.Diagnostics.Append(setPrivateWOHash(ctx, resp.Private, "oidc_client_secret_wo_hash", newSecretHash)...)
			if resp.Diagnostics.HasError() {
				return
			}
		} else {
			plan.Config.OIDCClientSecretWOVer = types.Int64Null()
		}

		if plan.Config.OIDCClientAssertion != nil && plan.Config.OIDCClientAssertion.PrivateKey != nil &&
			configData.Config.OIDCClientAssertion != nil && configData.Config.OIDCClientAssertion.PrivateKey != nil {
			if !configData.Config.OIDCClientAssertion.PrivateKey.PemKeyWO.IsNull() {
				newPKHash := calculateWOHash(configData.Config.OIDCClientAssertion.PrivateKey.PemKeyWO.ValueString())
				resp.Diagnostics.Append(setPrivateWOHash(ctx, resp.Private, "pem_key_wo_hash", newPKHash)...)
				if resp.Diagnostics.HasError() {
					return
				}
			} else {
				plan.Config.OIDCClientAssertion.PrivateKey.PemKeyWOVer = types.Int64Null()
			}
		}
	}

	authMethod, err := modelToAPI(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error building ACL auth method", err.Error())
		return
	}

	tflog.Debug(ctx, "Updating ACL auth method", map[string]interface{}{"name": authMethod.Name})
	_, _, err = r.providerConfig.Client().ACLAuthMethods().Update(authMethod, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error updating ACL auth method", err.Error())
		return
	}
	tflog.Debug(ctx, "Updated ACL auth method", map[string]interface{}{"name": authMethod.Name})

	fetched, _, err := r.providerConfig.Client().ACLAuthMethods().Get(authMethod.Name, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error reading ACL auth method after update", err.Error())
		return
	}

	var planClientSecret, planPemKey string
	if plan.Config != nil {
		planClientSecret = plan.Config.OIDCClientSecret.ValueString()
		if plan.Config.OIDCClientAssertion != nil && plan.Config.OIDCClientAssertion.PrivateKey != nil {
			planPemKey = plan.Config.OIDCClientAssertion.PrivateKey.PemKey.ValueString()
		}
	}

	apiToModel(ctx, fetched, &plan)

	if plan.Config != nil {
		if planClientSecret != "" {
			plan.Config.OIDCClientSecret = types.StringValue(planClientSecret)
		}
		if plan.Config.OIDCClientAssertion != nil && plan.Config.OIDCClientAssertion.PrivateKey != nil && planPemKey != "" {
			plan.Config.OIDCClientAssertion.PrivateKey.PemKey = types.StringValue(planPemKey)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ACLAuthMethodResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data aclAuthMethodModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	tflog.Debug(ctx, "Deleting ACL auth method", map[string]interface{}{"name": name})
	_, err := r.providerConfig.Client().ACLAuthMethods().Delete(name, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting ACL auth method", err.Error())
		return
	}
	tflog.Debug(ctx, "Deleted ACL auth method", map[string]interface{}{"name": name})
}

func (r *ACLAuthMethodResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func (r *ACLAuthMethodResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan aclAuthMethodModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Config == nil {
		return
	}

	var configData aclAuthMethodModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &configData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if configData.Config == nil {
		return
	}

	var state *aclAuthMethodModel
	if !req.State.Raw.IsNull() {
		var stateData aclAuthMethodModel
		resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
		if resp.Diagnostics.HasError() {
			return
		}
		state = &stateData
	}

	planModified := false

	versionExplicit := !configData.Config.OIDCClientSecretWOVer.IsNull()
	if configData.Config.OIDCClientSecretWO.IsNull() && !versionExplicit {
		desired := types.Int64Null()
		if state != nil && state.Config != nil {
			desired = state.Config.OIDCClientSecretWOVer
		}
		if !plan.Config.OIDCClientSecretWOVer.Equal(desired) {
			plan.Config.OIDCClientSecretWOVer = desired
			planModified = true
		}
	} else if !configData.Config.OIDCClientSecretWO.IsNull() && !versionExplicit {
		newSecretHash := calculateWOHash(configData.Config.OIDCClientSecretWO.ValueString())
		oldSecretHash, diags := getPrivateWOHash(ctx, req.Private, "oidc_client_secret_wo_hash")
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if newSecretHash != oldSecretHash {
			oldVer := int64(0)
			if state != nil && state.Config != nil && !state.Config.OIDCClientSecretWOVer.IsNull() {
				oldVer = state.Config.OIDCClientSecretWOVer.ValueInt64()
			}
			plan.Config.OIDCClientSecretWOVer = types.Int64Value(oldVer + 1)
		} else if state != nil && state.Config != nil {
			plan.Config.OIDCClientSecretWOVer = state.Config.OIDCClientSecretWOVer
		} else {
			plan.Config.OIDCClientSecretWOVer = types.Int64Value(1)
		}
		planModified = true
	}

	if configData.Config.OIDCClientAssertion != nil && configData.Config.OIDCClientAssertion.PrivateKey != nil &&
		plan.Config.OIDCClientAssertion != nil && plan.Config.OIDCClientAssertion.PrivateKey != nil {
		pkVersionExplicit := !configData.Config.OIDCClientAssertion.PrivateKey.PemKeyWOVer.IsNull()
		if configData.Config.OIDCClientAssertion.PrivateKey.PemKeyWO.IsNull() && !pkVersionExplicit {
			desired := types.Int64Null()
			if state != nil && state.Config != nil && state.Config.OIDCClientAssertion != nil &&
				state.Config.OIDCClientAssertion.PrivateKey != nil {
				desired = state.Config.OIDCClientAssertion.PrivateKey.PemKeyWOVer
			}
			if !plan.Config.OIDCClientAssertion.PrivateKey.PemKeyWOVer.Equal(desired) {
				plan.Config.OIDCClientAssertion.PrivateKey.PemKeyWOVer = desired
				planModified = true
			}
		} else if !configData.Config.OIDCClientAssertion.PrivateKey.PemKeyWO.IsNull() && !pkVersionExplicit {
			newPKHash := calculateWOHash(configData.Config.OIDCClientAssertion.PrivateKey.PemKeyWO.ValueString())
			oldPKHash, diags := getPrivateWOHash(ctx, req.Private, "pem_key_wo_hash")
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			if newPKHash != oldPKHash {
				oldPKVer := int64(0)
				if state != nil && state.Config != nil && state.Config.OIDCClientAssertion != nil &&
					state.Config.OIDCClientAssertion.PrivateKey != nil &&
					!state.Config.OIDCClientAssertion.PrivateKey.PemKeyWOVer.IsNull() {
					oldPKVer = state.Config.OIDCClientAssertion.PrivateKey.PemKeyWOVer.ValueInt64()
				}
				plan.Config.OIDCClientAssertion.PrivateKey.PemKeyWOVer = types.Int64Value(oldPKVer + 1)
			} else if state != nil && state.Config != nil && state.Config.OIDCClientAssertion != nil &&
				state.Config.OIDCClientAssertion.PrivateKey != nil {
				plan.Config.OIDCClientAssertion.PrivateKey.PemKeyWOVer = state.Config.OIDCClientAssertion.PrivateKey.PemKeyWOVer
			} else {
				plan.Config.OIDCClientAssertion.PrivateKey.PemKeyWOVer = types.Int64Value(1)
			}
			planModified = true
		}
	}

	if planModified {
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

func (r *ACLAuthMethodResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {StateUpgrader: upgradeStateV0toV1},
	}
}

func upgradeStateV0toV1(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	type v0PrivateKey struct {
		PemKey      string `json:"pem_key"`
		PemKeyFile  string `json:"pem_key_file"`
		PemCert     string `json:"pem_cert"`
		PemCertFile string `json:"pem_cert_file"`
		KeyIDHeader string `json:"key_id_header"`
		KeyID       string `json:"key_id"`
	}
	type v0ClientAssertion struct {
		Audience     []string          `json:"audience"`
		ExtraHeaders map[string]string `json:"extra_headers"`
		KeyAlgorithm string            `json:"key_algorithm"`
		KeySource    string            `json:"key_source"`
		PrivateKey   []v0PrivateKey    `json:"private_key"`
	}
	type v0Config struct {
		JWTValidationPubKeys []string            `json:"jwt_validation_pub_keys"`
		JWKSURL              string              `json:"jwks_url"`
		JWKSCACert           string              `json:"jwks_ca_cert"`
		OIDCDiscoveryURL     string              `json:"oidc_discovery_url"`
		OIDCClientID         string              `json:"oidc_client_id"`
		OIDCClientSecret     string              `json:"oidc_client_secret"`
		OIDCClientAssertion  []v0ClientAssertion `json:"oidc_client_assertion"`
		OIDCEnablePKCE       bool                `json:"oidc_enable_pkce"`
		OIDCDisableUserInfo  bool                `json:"oidc_disable_userinfo"`
		OIDCScopes           []string            `json:"oidc_scopes"`
		BoundAudiences       []string            `json:"bound_audiences"`
		BoundIssuer          []string            `json:"bound_issuer"`
		AllowedRedirectURIs  []string            `json:"allowed_redirect_uris"`
		DiscoveryCAPEM       []string            `json:"discovery_ca_pem"`
		SigningAlgs          []string            `json:"signing_algs"`
		ExpirationLeeway     string              `json:"expiration_leeway"`
		NotBeforeLeeway      string              `json:"not_before_leeway"`
		ClockSkewLeeway      string              `json:"clock_skew_leeway"`
		ClaimMappings        map[string]string   `json:"claim_mappings"`
		ListClaimMappings    map[string]string   `json:"list_claim_mappings"`
		VerboseLogging       bool                `json:"verbose_logging"`
	}
	type v0State struct {
		ID              string     `json:"id"`
		Name            string     `json:"name"`
		Type            string     `json:"type"`
		TokenLocality   string     `json:"token_locality"`
		MaxTokenTTL     string     `json:"max_token_ttl"`
		TokenNameFormat string     `json:"token_name_format"`
		Default         bool       `json:"default"`
		Config          []v0Config `json:"config"`
	}

	var old v0State
	if err := json.Unmarshal(req.RawState.JSON, &old); err != nil {
		resp.Diagnostics.AddError("Failed to parse prior state for upgrade", err.Error())
		return
	}

	maxTTL := old.MaxTokenTTL
	if maxTTL != "" {
		if d, err := time.ParseDuration(maxTTL); err == nil {
			maxTTL = d.String()
		}
	}
	newData := aclAuthMethodModel{
		ID:              types.StringValue(old.ID),
		Name:            types.StringValue(old.Name),
		Type:            types.StringValue(old.Type),
		TokenLocality:   types.StringValue(old.TokenLocality),
		MaxTokenTTL:     types.StringValue(maxTTL),
		TokenNameFormat: types.StringValue(old.TokenNameFormat),
		Default:         types.BoolValue(old.Default),
	}

	if len(old.Config) > 0 {
		c := old.Config[0]
		cfg := &aclAuthMethodConfigModel{
			JWTValidationPubKeys:  optionalListFromAPI(c.JWTValidationPubKeys),
			JWKSURL:               optionalStringFromAPI(c.JWKSURL),
			JWKSCACert:            optionalStringFromAPI(c.JWKSCACert),
			OIDCDiscoveryURL:      optionalStringFromAPI(c.OIDCDiscoveryURL),
			OIDCClientID:          optionalStringFromAPI(c.OIDCClientID),
			OIDCClientSecret:      optionalStringFromAPI(c.OIDCClientSecret),
			OIDCClientSecretWO:    types.StringNull(),
			OIDCClientSecretWOVer: types.Int64Null(),
			OIDCEnablePKCE:        types.BoolValue(c.OIDCEnablePKCE),
			OIDCDisableUserInfo:   types.BoolValue(c.OIDCDisableUserInfo),
			OIDCScopes:            optionalListFromAPI(c.OIDCScopes),
			BoundAudiences:        optionalListFromAPI(c.BoundAudiences),
			BoundIssuer:           optionalListFromAPI(c.BoundIssuer),
			AllowedRedirectURIs:   optionalListFromAPI(c.AllowedRedirectURIs),
			DiscoveryCAPEM:        optionalListFromAPI(c.DiscoveryCAPEM),
			SigningAlgs:           optionalListFromAPI(c.SigningAlgs),
			ExpirationLeeway:      types.StringValue(c.ExpirationLeeway),
			NotBeforeLeeway:       types.StringValue(c.NotBeforeLeeway),
			ClockSkewLeeway:       types.StringValue(c.ClockSkewLeeway),
			ClaimMappings:         optionalMapFromAPI(c.ClaimMappings),
			ListClaimMappings:     optionalMapFromAPI(c.ListClaimMappings),
			VerboseLogging:        types.BoolValue(c.VerboseLogging),
		}

		if len(c.OIDCClientAssertion) > 0 {
			a := c.OIDCClientAssertion[0]
			ass := &aclAuthMethodClientAssertionModel{
				Audience:     optionalListFromAPI(a.Audience),
				ExtraHeaders: optionalMapFromAPI(a.ExtraHeaders),
				KeyAlgorithm: optionalStringFromAPI(a.KeyAlgorithm),
				KeySource:    optionalStringFromAPI(a.KeySource),
			}
			if len(a.PrivateKey) > 0 {
				pk := a.PrivateKey[0]
				ass.PrivateKey = &aclAuthMethodPrivateKeyModel{
					PemKey:      optionalStringFromAPI(pk.PemKey),
					PemKeyWO:    types.StringNull(),
					PemKeyWOVer: types.Int64Null(),
					PemKeyFile:  optionalStringFromAPI(pk.PemKeyFile),
					PemCert:     optionalStringFromAPI(pk.PemCert),
					PemCertFile: optionalStringFromAPI(pk.PemCertFile),
					KeyIDHeader: types.StringValue(pk.KeyIDHeader),
					KeyID:       optionalStringFromAPI(pk.KeyID),
				}
			}
			cfg.OIDCClientAssertion = ass
		}

		newData.Config = cfg
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newData)...)
}

func calculateWOHash(value string) string {
	h := sha256.New()
	h.Write([]byte(value))
	return hex.EncodeToString(h.Sum(nil))
}

type privateHashWrapper struct {
	Hash string `json:"hash"`
}

type privateStateSetter interface {
	SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics
}

type privateStateGetter interface {
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
}

func setPrivateWOHash(ctx context.Context, private privateStateSetter, key, hash string) diag.Diagnostics {
	data, err := json.Marshal(privateHashWrapper{Hash: hash})
	if err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Error marshaling hash", err.Error())}
	}
	return private.SetKey(ctx, key, data)
}

func getPrivateWOHash(ctx context.Context, private privateStateGetter, key string) (string, diag.Diagnostics) {
	data, diags := private.GetKey(ctx, key)
	if diags.HasError() {
		return "", diags
	}
	if len(data) == 0 {
		return "", nil
	}
	var wrapper privateHashWrapper
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return "", diag.Diagnostics{diag.NewErrorDiagnostic("Error unmarshaling hash", err.Error())}
	}
	return wrapper.Hash, nil
}

func modelToAPI(ctx context.Context, data *aclAuthMethodModel) (*api.ACLAuthMethod, error) {
	am := &api.ACLAuthMethod{
		Name:            data.Name.ValueString(),
		Type:            data.Type.ValueString(),
		TokenLocality:   data.TokenLocality.ValueString(),
		TokenNameFormat: data.TokenNameFormat.ValueString(),
		Default:         data.Default.ValueBool(),
	}

	if ttlStr := data.MaxTokenTTL.ValueString(); ttlStr != "" {
		ttl, err := time.ParseDuration(ttlStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse max_token_ttl %q: %w", ttlStr, err)
		}
		am.MaxTokenTTL = ttl
	}

	if data.Config != nil {
		cfg, err := configModelToAPI(ctx, data.Config)
		if err != nil {
			return nil, err
		}
		am.Config = cfg
	}

	return am, nil
}

func configModelToAPI(ctx context.Context, m *aclAuthMethodConfigModel) (*api.ACLAuthMethodConfig, error) {
	cfg := &api.ACLAuthMethodConfig{
		JWKSURL:             m.JWKSURL.ValueString(),
		JWKSCACert:          m.JWKSCACert.ValueString(),
		OIDCDiscoveryURL:    m.OIDCDiscoveryURL.ValueString(),
		OIDCClientID:        m.OIDCClientID.ValueString(),
		OIDCEnablePKCE:      m.OIDCEnablePKCE.ValueBool(),
		OIDCDisableUserInfo: m.OIDCDisableUserInfo.ValueBool(),
		VerboseLogging:      m.VerboseLogging.ValueBool(),
	}

	if !m.OIDCClientSecretWO.IsNull() && !m.OIDCClientSecretWO.IsUnknown() {
		cfg.OIDCClientSecret = m.OIDCClientSecretWO.ValueString()
	} else {
		cfg.OIDCClientSecret = m.OIDCClientSecret.ValueString()
	}

	if !m.JWTValidationPubKeys.IsNull() {
		var keys []string
		if diags := m.JWTValidationPubKeys.ElementsAs(ctx, &keys, false); diags.HasError() {
			return nil, fmt.Errorf("reading jwt_validation_pub_keys")
		}
		cfg.JWTValidationPubKeys = keys
	}
	if !m.OIDCScopes.IsNull() {
		var vals []string
		if diags := m.OIDCScopes.ElementsAs(ctx, &vals, false); diags.HasError() {
			return nil, fmt.Errorf("reading oidc_scopes")
		}
		cfg.OIDCScopes = vals
	}
	if !m.BoundAudiences.IsNull() {
		var vals []string
		if diags := m.BoundAudiences.ElementsAs(ctx, &vals, false); diags.HasError() {
			return nil, fmt.Errorf("reading bound_audiences")
		}
		cfg.BoundAudiences = vals
	}
	if !m.BoundIssuer.IsNull() {
		var vals []string
		if diags := m.BoundIssuer.ElementsAs(ctx, &vals, false); diags.HasError() {
			return nil, fmt.Errorf("reading bound_issuer")
		}
		cfg.BoundIssuer = vals
	}
	if !m.AllowedRedirectURIs.IsNull() {
		var vals []string
		if diags := m.AllowedRedirectURIs.ElementsAs(ctx, &vals, false); diags.HasError() {
			return nil, fmt.Errorf("reading allowed_redirect_uris")
		}
		cfg.AllowedRedirectURIs = vals
	}
	if !m.DiscoveryCAPEM.IsNull() {
		var vals []string
		if diags := m.DiscoveryCAPEM.ElementsAs(ctx, &vals, false); diags.HasError() {
			return nil, fmt.Errorf("reading discovery_ca_pem")
		}
		cfg.DiscoveryCaPem = vals
	}
	if !m.SigningAlgs.IsNull() {
		var vals []string
		if diags := m.SigningAlgs.ElementsAs(ctx, &vals, false); diags.HasError() {
			return nil, fmt.Errorf("reading signing_algs")
		}
		cfg.SigningAlgs = vals
	}
	if !m.ClaimMappings.IsNull() {
		var vals map[string]string
		if diags := m.ClaimMappings.ElementsAs(ctx, &vals, false); diags.HasError() {
			return nil, fmt.Errorf("reading claim_mappings")
		}
		cfg.ClaimMappings = vals
	}
	if !m.ListClaimMappings.IsNull() {
		var vals map[string]string
		if diags := m.ListClaimMappings.ElementsAs(ctx, &vals, false); diags.HasError() {
			return nil, fmt.Errorf("reading list_claim_mappings")
		}
		cfg.ListClaimMappings = vals
	}
	if leeway := m.ExpirationLeeway.ValueString(); leeway != "" && leeway != "0s" {
		dur, err := time.ParseDuration(leeway)
		if err != nil {
			return nil, fmt.Errorf("parsing expiration_leeway: %w", err)
		}
		cfg.ExpirationLeeway = dur
	}
	if leeway := m.NotBeforeLeeway.ValueString(); leeway != "" && leeway != "0s" {
		dur, err := time.ParseDuration(leeway)
		if err != nil {
			return nil, fmt.Errorf("parsing not_before_leeway: %w", err)
		}
		cfg.NotBeforeLeeway = dur
	}
	if leeway := m.ClockSkewLeeway.ValueString(); leeway != "" && leeway != "0s" {
		dur, err := time.ParseDuration(leeway)
		if err != nil {
			return nil, fmt.Errorf("parsing clock_skew_leeway: %w", err)
		}
		cfg.ClockSkewLeeway = dur
	}

	if m.OIDCClientAssertion != nil {
		cAss := &api.OIDCClientAssertion{
			KeyAlgorithm: m.OIDCClientAssertion.KeyAlgorithm.ValueString(),
			KeySource:    api.OIDCClientAssertionKeySource(m.OIDCClientAssertion.KeySource.ValueString()),
		}
		if !m.OIDCClientAssertion.Audience.IsNull() && !m.OIDCClientAssertion.Audience.IsUnknown() {
			var aud []string
			if diags := m.OIDCClientAssertion.Audience.ElementsAs(ctx, &aud, false); diags.HasError() {
				return nil, fmt.Errorf("reading audience")
			}
			cAss.Audience = aud
		}
		if !m.OIDCClientAssertion.ExtraHeaders.IsNull() && !m.OIDCClientAssertion.ExtraHeaders.IsUnknown() {
			var headers map[string]string
			if diags := m.OIDCClientAssertion.ExtraHeaders.ElementsAs(ctx, &headers, false); diags.HasError() {
				return nil, fmt.Errorf("reading extra_headers")
			}
			cAss.ExtraHeaders = headers
		}
		if m.OIDCClientAssertion.PrivateKey != nil {
			pk := &api.OIDCClientAssertionKey{
				PemKeyFile:  m.OIDCClientAssertion.PrivateKey.PemKeyFile.ValueString(),
				PemCert:     m.OIDCClientAssertion.PrivateKey.PemCert.ValueString(),
				PemCertFile: m.OIDCClientAssertion.PrivateKey.PemCertFile.ValueString(),
				KeyID:       m.OIDCClientAssertion.PrivateKey.KeyID.ValueString(),
				KeyIDHeader: api.OIDCClientAssertionKeyIDHeader(m.OIDCClientAssertion.PrivateKey.KeyIDHeader.ValueString()),
			}
			if !m.OIDCClientAssertion.PrivateKey.PemKeyWO.IsNull() && !m.OIDCClientAssertion.PrivateKey.PemKeyWO.IsUnknown() {
				pk.PemKey = m.OIDCClientAssertion.PrivateKey.PemKeyWO.ValueString()
			} else {
				pk.PemKey = m.OIDCClientAssertion.PrivateKey.PemKey.ValueString()
			}
			cAss.PrivateKey = pk
		}
		cfg.OIDCClientAssertion = cAss
	}

	return cfg, nil
}

func apiToModel(ctx context.Context, am *api.ACLAuthMethod, data *aclAuthMethodModel) {
	data.ID = types.StringValue(am.Name)
	data.Name = types.StringValue(am.Name)
	data.Type = types.StringValue(am.Type)
	data.TokenLocality = types.StringValue(am.TokenLocality)
	data.TokenNameFormat = types.StringValue(am.TokenNameFormat)
	data.Default = types.BoolValue(am.Default)

	apiTTL := am.MaxTokenTTL.String()
	if existing := data.MaxTokenTTL.ValueString(); existing != "" {
		if existingDur, err := time.ParseDuration(existing); err != nil || existingDur != am.MaxTokenTTL {
			data.MaxTokenTTL = types.StringValue(apiTTL)
		}
	} else {
		data.MaxTokenTTL = types.StringValue(apiTTL)
	}

	if am.Config == nil {
		data.Config = nil
		return
	}

	if data.Config == nil {
		data.Config = &aclAuthMethodConfigModel{}
	}

	cfg := am.Config

	data.Config.JWKSURL = optionalStringFromAPI(cfg.JWKSURL)
	data.Config.JWKSCACert = optionalStringFromAPI(cfg.JWKSCACert)
	data.Config.OIDCDiscoveryURL = optionalStringFromAPI(cfg.OIDCDiscoveryURL)
	data.Config.OIDCClientID = optionalStringFromAPI(cfg.OIDCClientID)

	data.Config.OIDCEnablePKCE = types.BoolValue(cfg.OIDCEnablePKCE)
	data.Config.OIDCDisableUserInfo = types.BoolValue(cfg.OIDCDisableUserInfo)
	data.Config.VerboseLogging = types.BoolValue(cfg.VerboseLogging)
	data.Config.ExpirationLeeway = types.StringValue(cfg.ExpirationLeeway.String())
	data.Config.NotBeforeLeeway = types.StringValue(cfg.NotBeforeLeeway.String())
	data.Config.ClockSkewLeeway = types.StringValue(cfg.ClockSkewLeeway.String())

	data.Config.JWTValidationPubKeys = optionalListFromAPI(cfg.JWTValidationPubKeys)
	data.Config.OIDCScopes = optionalListFromAPI(cfg.OIDCScopes)
	data.Config.BoundAudiences = optionalListFromAPI(cfg.BoundAudiences)
	data.Config.BoundIssuer = optionalListFromAPI(cfg.BoundIssuer)
	data.Config.AllowedRedirectURIs = optionalListFromAPI(cfg.AllowedRedirectURIs)
	data.Config.DiscoveryCAPEM = optionalListFromAPI(cfg.DiscoveryCaPem)
	data.Config.SigningAlgs = optionalListFromAPI(cfg.SigningAlgs)
	data.Config.ClaimMappings = optionalMapFromAPI(cfg.ClaimMappings)
	data.Config.ListClaimMappings = optionalMapFromAPI(cfg.ListClaimMappings)

	if cfg.OIDCClientAssertion != nil {
		if data.Config.OIDCClientAssertion == nil {
			data.Config.OIDCClientAssertion = &aclAuthMethodClientAssertionModel{}
		}
		cAss := cfg.OIDCClientAssertion
		data.Config.OIDCClientAssertion.KeyAlgorithm = optionalStringFromAPI(cAss.KeyAlgorithm)
		data.Config.OIDCClientAssertion.KeySource = optionalStringFromAPI(string(cAss.KeySource))
		data.Config.OIDCClientAssertion.Audience = stringSliceToList(cAss.Audience)
		data.Config.OIDCClientAssertion.ExtraHeaders = optionalMapFromAPI(cAss.ExtraHeaders)

		if cAss.PrivateKey != nil {
			if data.Config.OIDCClientAssertion.PrivateKey == nil {
				data.Config.OIDCClientAssertion.PrivateKey = &aclAuthMethodPrivateKeyModel{}
			}
			pk := cAss.PrivateKey
			data.Config.OIDCClientAssertion.PrivateKey.PemKeyFile = optionalStringFromAPI(pk.PemKeyFile)
			data.Config.OIDCClientAssertion.PrivateKey.PemCert = optionalStringFromAPI(pk.PemCert)
			data.Config.OIDCClientAssertion.PrivateKey.PemCertFile = optionalStringFromAPI(pk.PemCertFile)
			data.Config.OIDCClientAssertion.PrivateKey.KeyID = optionalStringFromAPI(pk.KeyID)
			data.Config.OIDCClientAssertion.PrivateKey.KeyIDHeader = types.StringValue(string(pk.KeyIDHeader))
		} else {
			data.Config.OIDCClientAssertion.PrivateKey = nil
		}
	} else {
		data.Config.OIDCClientAssertion = nil
	}
}

func optionalStringFromAPI(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

func optionalListFromAPI(vals []string) types.List {
	if len(vals) == 0 {
		return types.ListNull(types.StringType)
	}
	return stringSliceToList(vals)
}

func optionalMapFromAPI(m map[string]string) types.Map {
	if len(m) == 0 {
		return types.MapNull(types.StringType)
	}
	return stringMapToTFMap(m)
}

func stringSliceToList(vals []string) types.List {
	elements := make([]attr.Value, len(vals))
	for i, v := range vals {
		elements[i] = types.StringValue(v)
	}
	return types.ListValueMust(types.StringType, elements)
}

func stringMapToTFMap(m map[string]string) types.Map {
	elements := make(map[string]attr.Value, len(m))
	for k, v := range m {
		elements[k] = types.StringValue(v)
	}
	return types.MapValueMust(types.StringType, elements)
}

type maxTokenTTLPlanModifier struct{}

func (m maxTokenTTLPlanModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	newDur, errNew := time.ParseDuration(req.PlanValue.ValueString())
	if errNew != nil {
		return
	}
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	oldDur, errOld := time.ParseDuration(req.StateValue.ValueString())
	if errOld != nil {
		return
	}
	if oldDur == newDur {
		resp.PlanValue = req.StateValue
	}
}

func (m maxTokenTTLPlanModifier) Description(_ context.Context) string {
	return "Suppresses duration format differences (e.g. '10m' vs '10m0s')."
}

func (m maxTokenTTLPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}
