import { ProjectIntegration } from 'app/model/integration.model';
import { UIArtifact, WorkflowRunResult, WorkflowRunResultArtifact, WorkflowRunResultArtifactManager, WorkflowRunResultStaticFile } from 'app/model/workflow.run.model';

export class WorkflowHelper {
    static toUIArtifact(results: Array<WorkflowRunResult>, artifactManagerIntegration?: ProjectIntegration): Array<UIArtifact> {
        if (!results) {
            return [];
        }

        let integrationArtifactManagerURL = '';
        if (artifactManagerIntegration) {
            integrationArtifactManagerURL = artifactManagerIntegration.config['url']?.value;
        }

        return results.map(r => {
            switch (r.type) {
                case 'artifact':
                case 'coverage':
                    let data = <WorkflowRunResultArtifact>r.data;
                    let uiArtifact = new UIArtifact();
                    uiArtifact.link = `./cdscdn/item/run-result/${data.cdn_hash}/download`;
                    uiArtifact.md5 = data.md5;
                    uiArtifact.name = data.name;
                    uiArtifact.size = data.size;
                    uiArtifact.human_size = this.getHumainFileSize(data.size);
                    uiArtifact.type = r.type === 'artifact' ? 'file' : r.type;
                    uiArtifact.file_type = uiArtifact.type;
                    return uiArtifact;
                case 'artifact-manager':
                    let dataAM = <WorkflowRunResultArtifactManager>r.data;
                    let uiArtifactAM = new UIArtifact();
                    uiArtifactAM.link = `${integrationArtifactManagerURL}${dataAM.repository_name}/${dataAM.path}`;
                    uiArtifactAM.md5 = dataAM.md5;
                    uiArtifactAM.name = dataAM.name;
                    uiArtifactAM.size = dataAM.size;
                    uiArtifactAM.human_size = this.getHumainFileSize(dataAM.size);
                    uiArtifactAM.type = dataAM.repository_type;
                    uiArtifactAM.file_type = dataAM.file_type ? dataAM.file_type : dataAM.repository_type;
                    return uiArtifactAM;
                case 'static-file':
                    let dataSF = <WorkflowRunResultStaticFile>r.data;
                    let uiArtifactSF = new UIArtifact();
                    uiArtifactSF.link = dataSF.remote_url;
                    uiArtifactSF.name = dataSF.name;
                    uiArtifactSF.type = 'static file';
                    return uiArtifactSF;
            }
        });
    }

    static getHumainFileSize(size: number): string {
        if (!size) {
            return '';
        }
        let i = Math.floor(Math.log(size) / Math.log(1024));
        let hSize = (size / Math.pow(1024, i)).toFixed(2);
        return hSize + ' ' + ['B', 'kB', 'MB', 'GB', 'TB'][i];
    }
}
