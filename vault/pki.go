package vault

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type CertOptions struct {
	CN                string `json:"common_name"`
	TTL               string `json:"ttl,omitempty"`
	AltNames          string `json:"alt_names,omitempty"`
	IPSans            string `json:"ip_sans,omitempty"`
	ExcludeCNFromSans bool   `json:"exclude_cn_from_sans,omitempty"`
}

func (v *Vault) CheckPKIBackend(backend string) error {
	if mounted, _ := v.IsMounted("pki", backend); !mounted {
		return fmt.Errorf("The PKI backend `%s` has not been configured. Try running `safe pki init --backend %s`\n", backend, backend)
	}
	return nil
}

func (v *Vault) RetrievePem(backend, path string) ([]byte, error) {
	if err := v.CheckPKIBackend(backend); err != nil {
		return nil, err
	}

	res, err := v.Curl("GET", fmt.Sprintf("/%s/%s/pem", backend, path), nil)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, DecodeErrorResponse(body)
	}

	return body, nil
}

func (v *Vault) CreateSignedCertificate(backend, role, path string, params CertOptions, skipIfExists bool) error {
	if err := v.CheckPKIBackend(backend); err != nil {
		return err
	}

	data, err := json.Marshal(params)
	if err != nil {
		return err
	}
	res, err := v.Curl("POST", fmt.Sprintf("%s/issue/%s", backend, role), data)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode >= 400 {
		return fmt.Errorf("Unable to create certificate %s: %s\n", params.CN, DecodeErrorResponse(body))
	}

	var raw map[string]interface{}
	if err = json.Unmarshal(body, &raw); err == nil {
		if d, ok := raw["data"]; ok {
			if data, ok := d.(map[string]interface{}); ok {
				var cert, key, serial string
				var c, k, s interface{}
				var ok bool
				if c, ok = data["certificate"]; !ok {
					return fmt.Errorf("No certificate found when issuing certificate %s:\n%v\n", params.CN, data)
				}
				if cert, ok = c.(string); !ok {
					return fmt.Errorf("Invalid data type for certificate %s:\n%v\n", params.CN, data)
				}
				if k, ok = data["private_key"]; !ok {
					return fmt.Errorf("No private_key found when issuing certificate %s:\n%v\n", params.CN, data)
				}
				if key, ok = k.(string); !ok {
					return fmt.Errorf("Invalid data type for private_key %s:\n%v\n", params.CN, data)
				}
				if s, ok = data["serial_number"]; !ok {
					return fmt.Errorf("No serial_number found when issuing certificate %s:\n%v\n", params.CN, data)
				}
				if serial, ok = s.(string); !ok {
					return fmt.Errorf("Invalid data type for serial_number %s:\n%v\n", params.CN, data)
				}

				secret, err := v.Read(path)
				if err != nil && !IsNotFound(err) {
					return err
				}
				err = secret.Set("cert", cert, skipIfExists)
				if err != nil {
					return err
				}
				err = secret.Set("key", key, skipIfExists)
				if err != nil {
					return err
				}
				err = secret.Set("combined", cert+key, skipIfExists)
				if err != nil {
					return err
				}
				err = secret.Set("serial", serial, skipIfExists)
				if err != nil {
					return err
				}
				return v.Write(path, secret)
			} else {
				return fmt.Errorf("Invalid response datatype requesting certificate %s:\n%v\n", params.CN, d)
			}
		} else {
			return fmt.Errorf("No data found when requesting certificate %s:\n%v\n", params.CN, d)
		}
	} else {
		return fmt.Errorf("Unparseable json creating certificate %s:\n%s\n", params.CN, body)
	}
}

func (v *Vault) RevokeCertificate(backend, serial string) error {
	if err := v.CheckPKIBackend(backend); err != nil {
		return err
	}

	if strings.ContainsRune(serial, '/') {
		secret, err := v.Read(serial)
		if err != nil {
			return err
		}
		if !secret.Has("serial") {
			return fmt.Errorf("Certificate specified using path %s, but no serial secret was found there", serial)
		}
		serial = secret.Get("serial")
	}

	d := struct {
		Serial string `json:"serial_number"`
	}{Serial: serial}

	data, err := json.Marshal(d)
	if err != nil {
		return err
	}

	res, err := v.Curl("POST", fmt.Sprintf("%s/revoke", backend), data)
	if err != nil {
		return err
	}

	if res.StatusCode >= 400 {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("Unable to revoke certificate %s: %s\n", serial, DecodeErrorResponse(body))
	}
	return nil
}

func (v *Vault) FindSigningCA(cert *X509, certPath string, signPath string) (*X509, string, error) {
	/* find the CA */
	if signPath != "" {
		if certPath == signPath {
			return cert, certPath, nil
		} else {
			s, err := v.Read(signPath)
			if err != nil {
				return nil, "", err
			}
			ca, err := s.X509(true)
			if err != nil {
				return nil, "", err
			}
			return ca, signPath, nil
		}
	} else {
		// Check if this cert is self-signed If so, don't change the value
		// of s, because its already the cert we loaded in. #Hax
		err := cert.Certificate.CheckSignature(
			cert.Certificate.SignatureAlgorithm,
			cert.Certificate.RawTBSCertificate,
			cert.Certificate.Signature,
		)
		if err == nil {
			return cert, certPath, nil
		} else {
			// Lets see if we can guess the CA if none was provided
			caPath := certPath[0:strings.LastIndex(certPath, "/")] + "/ca"
			s, err := v.Read(caPath)
			if err != nil {
				return nil, "", fmt.Errorf("No signing authority provided and no 'ca' sibling found")
			}
			ca, err := s.X509(true)
			if err != nil {
				return nil, "", err
			}
			return ca, caPath, nil
		}
	}
}

func (v *Vault) SaveSealKeys(keys []string) {
	path := "secret/vault/seal/keys"
	s := NewSecret()
	for i, key := range keys {
		s.Set(fmt.Sprintf("key%d", i+1), key, false)
	}
	v.Write(path, s)
}
