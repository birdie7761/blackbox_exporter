# delete service if it exists
if (Get-Service blackbox_exporter -ErrorAction SilentlyContinue) {
  $service = Get-WmiObject -Class Win32_Service -Filter "name='blackbox_exporter'"
  $service.delete()
}
