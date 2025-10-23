# Generate self-signed TLS certificate for development/testing
# For production, use proper certificates from a trusted CA (Let's Encrypt, etc.)

Write-Host "Generating self-signed TLS certificate..." -ForegroundColor Cyan

$certPath = ".\certs"
if (-not (Test-Path $certPath)) {
    New-Item -ItemType Directory -Path $certPath | Out-Null
}

# Generate certificate valid for 365 days
$cert = New-SelfSignedCertificate `
    -DnsName "localhost", "127.0.0.1" `
    -CertStoreLocation "Cert:\CurrentUser\My" `
    -NotAfter (Get-Date).AddDays(365) `
    -KeyAlgorithm RSA `
    -KeyLength 2048 `
    -KeyUsage DigitalSignature, KeyEncipherment `
    -TextExtension @("2.5.29.37={text}1.3.6.1.5.5.7.3.1")

$thumbprint = $cert.Thumbprint

# Export certificate to PEM format
$certPemPath = Join-Path $certPath "server.crt"
$keyPemPath = Join-Path $certPath "server.key"

# Export certificate
$certBytes = $cert.Export([System.Security.Cryptography.X509Certificates.X509ContentType]::Cert)
$certPem = "-----BEGIN CERTIFICATE-----`n"
$certPem += [System.Convert]::ToBase64String($certBytes, [System.Base64FormattingOptions]::InsertLineBreaks)
$certPem += "`n-----END CERTIFICATE-----"
$certPem | Out-File -FilePath $certPemPath -Encoding ASCII

# Export private key (Note: This is simplified for development)
# In production, manage keys securely with proper tools
$rsaKey = [System.Security.Cryptography.X509Certificates.RSACertificateExtensions]::GetRSAPrivateKey($cert)
$keyBytes = $rsaKey.ExportRSAPrivateKey()
$keyPem = "-----BEGIN RSA PRIVATE KEY-----`n"
$keyPem += [System.Convert]::ToBase64String($keyBytes, [System.Base64FormattingOptions]::InsertLineBreaks)
$keyPem += "`n-----END RSA PRIVATE KEY-----"
$keyPem | Out-File -FilePath $keyPemPath -Encoding ASCII

# Remove from certificate store
Get-ChildItem "Cert:\CurrentUser\My\$thumbprint" | Remove-Item

Write-Host "`n✅ Certificate generated successfully!" -ForegroundColor Green
Write-Host "   Certificate: $certPemPath" -ForegroundColor White
Write-Host "   Private Key: $keyPemPath" -ForegroundColor White
Write-Host "`n⚠️  WARNING: This is a self-signed certificate for development only!" -ForegroundColor Yellow
Write-Host "   For production, use certificates from a trusted Certificate Authority." -ForegroundColor Yellow
Write-Host "`nTo run with TLS:" -ForegroundColor Cyan
Write-Host "   `$env:TLS_ENABLED='true'" -ForegroundColor Gray
Write-Host "   `$env:TLS_CERT_FILE='$certPemPath'" -ForegroundColor Gray
Write-Host "   `$env:TLS_KEY_FILE='$keyPemPath'" -ForegroundColor Gray
Write-Host "   `$env:JWT_SECRET='your-secret-key'" -ForegroundColor Gray
Write-Host "   .\sentinel.exe`n" -ForegroundColor Gray
