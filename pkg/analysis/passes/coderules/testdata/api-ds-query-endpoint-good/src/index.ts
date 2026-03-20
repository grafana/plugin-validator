import { getDataSourceSrv } from "@grafana/runtime";

async function runQuery() {
  return getDataSourceSrv()
    .get("my-datasource")
    .then((datasource) => datasource.query({}));
}
