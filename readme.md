On windows you need to run the following to set the AWS Credentials 

`$env:AWS_ACCESS_KEY_ID = ""`
`$env:AWS_SECRET_ACCESS_KEY = ""`
`$env:AWS_REGION = "us-east-2"`
`$env:AWS_BUCKET = "90daysofdevops"`

Confirmation with 

`[System.Environment]::GetEnvironmentVariable('AWS_BUCKET')`