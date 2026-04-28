import { DataSourceWithBackend } from '@grafana/runtime';

export class DataSource extends DataSourceWithBackend {
  async testDatasource() {
    return {
      status: 'success',
      message: 'ok',
    };
  }
}
