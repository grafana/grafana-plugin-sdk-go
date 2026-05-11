package dsconfig

// Common field sets shared across many datasource schemas.
// These helpers reduce duplication when building config schemas
// programmatically. The generated JSON files remain self-contained.

// BasicAuthFields returns the standard basic-auth field set
// (toggle, username, password).
func BasicAuthFields() []ConfigField {
	return []ConfigField{
		{
			ID: "auth.basicAuth", Key: "basicAuth",
			Label: "Basic Auth", Description: "Enable basic authentication",
			ValueType: BooleanType, Target: RootTarget,
		},
		{
			ID: "auth.basicAuthUser", Key: "basicAuthUser",
			Label: "Username", ValueType: StringType, Target: RootTarget,
			DependsOn: "auth.basicAuth == true", RequiredWhen: "auth.basicAuth == true",
		},
		{
			ID: "auth.basicAuthPassword", Key: "basicAuthPassword",
			Label: "Password", ValueType: StringType, Target: SecureJSONTarget,
			SemanticType: PasswordType,
			DependsOn:    "auth.basicAuth == true",
		},
	}
}

// TLSFields returns the standard TLS/SSL field set.
func TLSFields() []ConfigField {
	return []ConfigField{
		{
			ID: "tls.tlsAuth", Key: "tlsAuth",
			Label: "TLS Client Authentication", ValueType: BooleanType, Target: JSONDataTarget,
		},
		{
			ID: "tls.tlsAuthWithCACert", Key: "tlsAuthWithCACert",
			Label: "With CA Cert", ValueType: BooleanType, Target: JSONDataTarget,
		},
		{
			ID: "tls.tlsSkipVerify", Key: "tlsSkipVerify",
			Label: "Skip TLS Verify", ValueType: BooleanType, Target: JSONDataTarget,
		},
		{
			ID: "tls.serverName", Key: "serverName",
			Label: "Server Name", ValueType: StringType, Target: JSONDataTarget,
			SemanticType: HostnameType,
		},
		{
			ID: "tls.tlsCACert", Key: "tlsCACert",
			Label: "CA Cert", ValueType: StringType, Target: SecureJSONTarget,
			DependsOn: "tls.tlsAuthWithCACert == true",
			UI:        &FieldUI{Component: UITextarea, Rows: 7},
		},
		{
			ID: "tls.tlsClientCert", Key: "tlsClientCert",
			Label: "Client Cert", ValueType: StringType, Target: SecureJSONTarget,
			DependsOn: "tls.tlsAuth == true",
			UI:        &FieldUI{Component: UITextarea, Rows: 7},
		},
		{
			ID: "tls.tlsClientKey", Key: "tlsClientKey",
			Label: "Client Key", ValueType: StringType, Target: SecureJSONTarget,
			DependsOn: "tls.tlsAuth == true",
			UI:        &FieldUI{Component: UITextarea, Rows: 7},
		},
	}
}

// CommonNetworkFields returns fields shared by many datasources:
// timeout, keepCookies, oauthPassThru, pdcInjected.
func CommonNetworkFields() []ConfigField {
	return []ConfigField{
		{
			ID: "network.timeout", Key: "timeout",
			Label: "Timeout", Description: "HTTP request timeout in seconds",
			ValueType: NumberType, Target: JSONDataTarget,
			Validations: []FieldValidationRule{
				{Type: RangeValidation, Min: ptr(1.0), Max: ptr(600.0)},
			},
		},
		{
			ID: "network.keepCookies", Key: "keepCookies",
			Label: "Allowed Cookies", Description: "Cookies to forward to the datasource",
			ValueType: ArrayType, Target: JSONDataTarget,
			Item: &FieldItemSchema{ValueType: StringType},
		},
		{
			ID: "network.oauthPassThru", Key: "oauthPassThru",
			Label:     "Forward OAuth Identity",
			ValueType: BooleanType, Target: JSONDataTarget,
		},
		{
			ID: "network.pdcInjected", Key: "pdcInjected",
			Label:     "Private Data Source Connect",
			ValueType: BooleanType, Target: JSONDataTarget,
		},
	}
}

// HTTPHeaderFields returns the standard custom HTTP headers field
// with indexedPair storage mapping.
func HTTPHeaderFields() []ConfigField {
	return []ConfigField{
		{
			ID: "httpHeaders", Key: "httpHeaders",
			Label: "Custom HTTP Headers", Description: "Additional headers sent with every request",
			ValueType: ArrayType, Target: JSONDataTarget,
			Item: &FieldItemSchema{
				ValueType: ObjectType,
				Fields: []ConfigField{
					{ID: "httpHeaders.item.name", Key: "name", Label: "Header Name", ValueType: StringType, IsItemField: ptr(true)},
					{ID: "httpHeaders.item.value", Key: "value", Label: "Header Value", ValueType: StringType, IsItemField: ptr(true)},
				},
			},
			Storage: &StorageMapping{
				Type:  IndexedPairMapping,
				Key:   &MappingField{Target: JSONDataTarget, Pattern: "httpHeaderName{index}"},
				Value: &MappingField{Target: SecureJSONTarget, Pattern: "httpHeaderValue{index}"},
			},
			Validations: []FieldValidationRule{
				{Type: ItemCountValidation, Max: ptr(10.0), Message: "Maximum 10 custom headers"},
			},
		},
	}
}

func ptr[T any](v T) *T { return &v }
