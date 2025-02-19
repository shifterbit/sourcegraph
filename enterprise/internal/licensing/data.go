package licensing

// The list of plans.
const (
	// PlanOldEnterpriseStarter is the old "Enterprise Starter" plan.
	PlanOldEnterpriseStarter Plan = "old-starter-0"
	// PlanOldEnterprise is the old "Enterprise" plan.
	PlanOldEnterprise Plan = "old-enterprise-0"

	// PlanTeam0 is the "Team" plan pre-4.0.
	PlanTeam0 Plan = "team-0"
	// PlanEnterprise0 is the "Enterprise" plan pre-4.0.
	PlanEnterprise0 Plan = "enterprise-0"

	// PlanBusiness0 is the "Business" plan for 4.0.
	PlanBusiness0 Plan = "business-0"
	// PlanEnterprise1 is the "Enterprise" plan for 4.0.
	PlanEnterprise1 Plan = "enterprise-1"

	// PlanFree0 is the default plan if no license key is set.
	PlanFree0 Plan = "free-0"
)

var allPlans = []Plan{
	PlanOldEnterpriseStarter,
	PlanOldEnterprise,
	PlanTeam0,
	PlanEnterprise0,

	PlanBusiness0,
	PlanEnterprise1,
	PlanFree0,
}

// The list of features. For each feature, add a new const here and the checking logic in
// isFeatureEnabled.
const (
	// FeatureSSO is whether non-builtin authentication may be used, such as GitHub
	// OAuth, GitLab OAuth, SAML, and OpenID.
	FeatureSSO Feature = "sso"

	// FeatureACLs is whether the Background Permissions Syncing may be be used for
	// setting repository permissions.
	FeatureACLs Feature = "acls"

	// FeatureExplicitPermissionsAPI is whether the Explicit Permissions API may be be used for
	// setting repository permissions.
	FeatureExplicitPermissionsAPI Feature = "explicit-permissions-api"

	// FeatureExtensionRegistry is whether publishing extensions to this Sourcegraph instance has been
	// purchased. If not, then extensions must be published to Sourcegraph.com. All instances may use
	// extensions published to Sourcegraph.com.
	FeatureExtensionRegistry Feature = "private-extension-registry"

	// FeatureRemoteExtensionsAllowDisallow is whether explicitly specify a list of allowed remote
	// extensions and prevent any other remote extensions from being used has been purchased. It
	// does not apply to locally published extensions.
	FeatureRemoteExtensionsAllowDisallow Feature = "remote-extensions-allow-disallow"

	// FeatureBranding is whether custom branding of this Sourcegraph instance has been purchased.
	FeatureBranding Feature = "branding"

	// FeatureCampaigns is whether campaigns (now: batch changes) on this Sourcegraph instance has been purchased.
	//
	// DEPRECATED: See FeatureBatchChanges.
	FeatureCampaigns Feature = "campaigns"

	// FeatureBatchChanges is whether Batch Changes on this Sourcegraph instance has been purchased.
	FeatureBatchChanges Feature = "batch-changes"

	// FeatureMonitoring is whether monitoring on this Sourcegraph instance has been purchased.
	FeatureMonitoring Feature = "monitoring"

	// FeatureBackupAndRestore is whether builtin backup and restore on this Sourcegraph instance
	// has been purchased.
	FeatureBackupAndRestore Feature = "backup-and-restore"

	// FeatureCodeInsights is whether Code Insights on this Sourcegraph instance has been purchased.
	FeatureCodeInsights Feature = "code-insights"
)

// planFeatures defines the features that are enabled for each plan.
var planFeatures = map[Plan][]Feature{
	PlanOldEnterpriseStarter: {},
	PlanOldEnterprise: {
		FeatureSSO,
		FeatureACLs,
		FeatureExplicitPermissionsAPI,
		FeatureExtensionRegistry,
		FeatureRemoteExtensionsAllowDisallow,
		FeatureBranding,
		FeatureCampaigns,
		FeatureBatchChanges,
		FeatureMonitoring,
		FeatureBackupAndRestore,
		FeatureCodeInsights,
	},
	PlanTeam0: {
		FeatureACLs,
		FeatureExplicitPermissionsAPI,
		FeatureSSO,
	},
	PlanEnterprise0: {
		FeatureACLs,
		FeatureExplicitPermissionsAPI,
		FeatureSSO,
	},

	PlanBusiness0: {
		FeatureACLs,
		FeatureCampaigns,
		FeatureBatchChanges,
		FeatureCodeInsights,
		FeatureSSO,
	},
	PlanEnterprise1: {
		FeatureACLs,
		FeatureCampaigns,
		FeatureCodeInsights,
		FeatureBatchChanges,
		FeatureExplicitPermissionsAPI,
		FeatureSSO,
	},
	PlanFree0: {
		FeatureSSO,
		FeatureMonitoring,
	},
}
