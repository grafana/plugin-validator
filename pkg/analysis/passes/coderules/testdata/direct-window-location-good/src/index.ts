import { locationService } from '@grafana/runtime';

function getParams() {
  const search = locationService.getSearch();
  const location = locationService.getLocation();
}
