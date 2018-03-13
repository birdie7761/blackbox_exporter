# delete service if it already exists
if (Get-Service blackbox_exporter -ErrorAction SilentlyContinue) {
  $service = Get-WmiObject -Class Win32_Service -Filter "name='blackbox_exporter'"
  $service.StopService()
  Start-Sleep -s 1
  $service.delete()
}

$workdir = Split-Path $MyInvocation.MyCommand.Path

# create new service
New-Service -name blackbox_exporter `
  -displayName blackbox_exporter `
  -binaryPathName "`"$workdir\\blackbox_exporter.exe`" --config.file=`"$workdir\\blackbox.yml`""
