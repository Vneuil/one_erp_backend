$headers = @{
    "Content-Type" = "application/json"
    "X-Internal-Token" = "adminseed123"
}

# 1. Create Company
$companyBody = @{
    code = "MANNA"
    name = "Manna ERP"
    email = "admin@manna-erp.com"
} | ConvertTo-Json

$companyResponse = Invoke-RestMethod -Uri "http://localhost:3000/internal/provisioning/companies" -Method Post -Headers $headers -Body $companyBody
$companyId = $companyResponse.data.id

Write-Output "Created company with ID: $companyId"

# 2. Activate Company (create admin user)
$activateBody = @{
    name = "Admin"
    email = "admin@manna-erp.com"
    passwordHash = "`$2a`$10`$h3M6IEKfKFSOlBI67.htSuL6GSQ4K/WD5z0ofbXO9B0kqPWf.CXcW"
} | ConvertTo-Json

$activateResponse = Invoke-RestMethod -Uri "http://localhost:3000/internal/provisioning/companies/$companyId/activate" -Method Post -Headers $headers -Body $activateBody

Write-Output "Created admin user:"
$activateResponse | ConvertTo-Json -Depth 10
