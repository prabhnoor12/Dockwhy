import * as vscode from 'vscode';
import * as cp from 'child_process';
import * as path from 'path';

let stoppedContainersProvider: ContainerTreeProvider;
let runningContainersProvider: ContainerTreeProvider;

export function activate(context: vscode.ExtensionContext) {
    stoppedContainersProvider = new ContainerTreeProvider(true);
    runningContainersProvider = new ContainerTreeProvider(false);

    vscode.window.registerTreeDataProvider('dockwhy.containers', stoppedContainersProvider);
    vscode.window.registerTreeDataProvider('dockwhy.running', runningContainersProvider);

    const diagnose = vscode.commands.registerCommand('dockwhy.diagnose', async () => {
        const containerName = await vscode.window.showInputBox({
            prompt: 'Enter container name or ID',
            placeHolder: 'my-container'
        });
        if (containerName) {
            await runDiagnosis(containerName, context);
        }
    });

    const diagnoseProject = vscode.commands.registerCommand('dockwhy.diagnoseProject', async () => {
        const projectName = await vscode.window.showInputBox({
            prompt: 'Enter Docker Compose project name',
            placeHolder: 'my-stack'
        });
        if (projectName) {
            await runDiagnosis(projectName, context, ['--project']);
        }
    });

    const diagnoseSelected = vscode.commands.registerCommand('dockwhy.diagnoseSelected', async (item: ContainerItem) => {
        if (item) {
            await runDiagnosis(item.containerName, context);
        }
    });

    const refreshContainers = vscode.commands.registerCommand('dockwhy.refreshContainers', () => {
        stoppedContainersProvider.refresh();
        runningContainersProvider.refresh();
    });

    const showTrend = vscode.commands.registerCommand('dockwhy.showTrend', async () => {
        const containerName = await vscode.window.showInputBox({
            prompt: 'Enter container name or ID for trend analysis',
            placeHolder: 'my-container'
        });
        if (containerName) {
            await runDiagnosis(containerName, context, ['--trend']);
        }
    });

    const showResources = vscode.commands.registerCommand('dockwhy.showResources', async () => {
        const containerName = await vscode.window.showInputBox({
            prompt: 'Enter container name or ID for resource analysis',
            placeHolder: 'my-container'
        });
        if (containerName) {
            await runDiagnosis(containerName, context, ['--resources']);
        }
    });

    const generateReport = vscode.commands.registerCommand('dockwhy.generateReport', async () => {
        const containerName = await vscode.window.showInputBox({
            prompt: 'Enter container name or ID',
            placeHolder: 'my-container'
        });
        if (containerName) {
            const savePath = await vscode.window.showSaveDialog({
                filters: { 'Markdown': ['md'] },
                defaultUri: vscode.Uri.file(`${containerName}-incident-report.md`)
            });
            if (savePath) {
                await runDiagnosis(containerName, context, ['--report', savePath.fsPath]);
                vscode.window.showInformationMessage(`Report saved to ${savePath.fsPath}`);
            }
        }
    });

    context.subscriptions.push(
        diagnose,
        diagnoseProject,
        diagnoseSelected,
        refreshContainers,
        showTrend,
        showResources,
        generateReport
    );

    const config = vscode.workspace.getConfiguration('dockwhy');
    if (config.get<boolean>('autoRefresh')) {
        const interval = config.get<number>('refreshInterval') || 30;
        setInterval(() => {
            stoppedContainersProvider.refresh();
            runningContainersProvider.refresh();
        }, interval * 1000);
    }
}

async function runDiagnosis(containerName: string, context: vscode.ExtensionContext, extraArgs: string[] = []) {
    const config = vscode.workspace.getConfiguration('dockwhy');
    const executablePath = config.get<string>('executablePath') || 'dockwhy';
    const tail = config.get<number>('defaultTail') || 50;
    const timeout = config.get<string>('defaultTimeout') || '10s';

    const args = [
        ...extraArgs,
        '--tail', tail.toString(),
        '--timeout', timeout,
        '--json',
        containerName
    ];

    await vscode.window.withProgress({
        location: vscode.ProgressLocation.Notification,
        title: `Diagnosing ${containerName}...`,
        cancellable: false
    }, async () => {
        try {
            const output = await executeCommand(executablePath, args);
            showDiagnosisPanel(containerName, output, context);
        } catch (error: any) {
            vscode.window.showErrorMessage(`Failed to diagnose ${containerName}: ${error.message}`);
        }
    });
}

function executeCommand(command: string, args: string[]): Promise<string> {
    return new Promise((resolve, reject) => {
        cp.execFile(command, args, { maxBuffer: 10 * 1024 * 1024 }, (error, stdout, stderr) => {
            if (error) {
                reject(new Error(stderr || error.message));
            } else {
                resolve(stdout);
            }
        });
    });
}

function showDiagnosisPanel(containerName: string, output: string, context: vscode.ExtensionContext) {
    const panel = vscode.window.createWebviewPanel(
        'dockwhyDiagnosis',
        `Dockwhy: ${containerName}`,
        vscode.ViewColumn.One,
        { enableScripts: true }
    );

    let diagnosis: any;
    try {
        diagnosis = JSON.parse(output);
    } catch {
        diagnosis = { raw: output };
    }

    panel.webview.html = getWebviewContent(containerName, diagnosis);
}

function getWebviewContent(containerName: string, diagnosis: any): string {
    if (diagnosis.raw) {
        return `<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: var(--vscode-font-family); padding: 20px; }
        pre { background: var(--vscode-editor-background); padding: 15px; border-radius: 5px; overflow-x: auto; }
    </style>
</head>
<body>
    <h1>Diagnosis for ${containerName}</h1>
    <pre>${diagnosis.raw}</pre>
</body>
</html>`;
    }

    const findings = diagnosis.findings || [];
    const findingsHtml = findings.map((f: any) => `
        <div class="finding severity-${f.severity || 'info'}">
            <h3>${f.title || 'Finding'}</h3>
            <div class="meta">
                <span class="badge severity">${f.severity || 'info'}</span>
                ${f.confidence ? `<span class="badge confidence">${f.confidence}% confidence</span>` : ''}
            </div>
            ${f.description ? `<p>${f.description}</p>` : ''}
            ${f.evidence ? `<div class="evidence"><strong>Evidence:</strong> ${f.evidence}</div>` : ''}
            ${f.advice ? `<div class="advice"><strong>Recommendation:</strong> ${f.advice}</div>` : ''}
        </div>
    `).join('');

    return `<!DOCTYPE html>
<html>
<head>
    <style>
        body {
            font-family: var(--vscode-font-family);
            padding: 20px;
            color: var(--vscode-foreground);
            background: var(--vscode-editor-background);
        }
        h1 { border-bottom: 2px solid var(--vscode-panel-border); padding-bottom: 10px; }
        .finding {
            background: var(--vscode-editorWidget-background);
            border-left: 4px solid var(--vscode-foreground);
            padding: 15px;
            margin: 15px 0;
            border-radius: 4px;
        }
        .finding.severity-critical { border-left-color: #f44336; }
        .finding.severity-high { border-left-color: #ff9800; }
        .finding.severity-medium { border-left-color: #ffeb3b; }
        .finding.severity-low { border-left-color: #4caf50; }
        .finding.severity-info { border-left-color: #2196f3; }
        .meta { margin: 10px 0; }
        .badge {
            display: inline-block;
            padding: 3px 8px;
            border-radius: 3px;
            font-size: 12px;
            margin-right: 8px;
            background: var(--vscode-badge-background);
            color: var(--vscode-badge-foreground);
        }
        .evidence, .advice {
            margin-top: 10px;
            padding: 10px;
            background: var(--vscode-textBlockQuote-background);
            border-radius: 3px;
        }
        .advice { border-left: 3px solid #4caf50; }
        .summary {
            background: var(--vscode-editorWidget-background);
            padding: 15px;
            border-radius: 4px;
            margin: 20px 0;
        }
        .summary-item { display: inline-block; margin-right: 20px; }
    </style>
</head>
<body>
    <h1>Diagnosis for ${containerName}</h1>
    
    ${diagnosis.container ? `
    <div class="summary">
        <div class="summary-item"><strong>State:</strong> ${diagnosis.container.state || 'unknown'}</div>
        <div class="summary-item"><strong>Status:</strong> ${diagnosis.container.status || 'unknown'}</div>
        ${diagnosis.container.exit_code !== undefined ? `<div class="summary-item"><strong>Exit Code:</strong> ${diagnosis.container.exit_code}</div>` : ''}
        ${diagnosis.container.image ? `<div class="summary-item"><strong>Image:</strong> ${diagnosis.container.image}</div>` : ''}
    </div>
    ` : ''}
    
    ${findings.length > 0 ? `
        <h2>Findings (${findings.length})</h2>
        ${findingsHtml}
    ` : '<p>No findings reported.</p>'}
    
    ${diagnosis.smart_logs ? `
        <h2>Smart Logs</h2>
        <pre>${diagnosis.smart_logs}</pre>
    ` : ''}
    
    ${diagnosis.resources ? `
        <h2>Resource Recommendations</h2>
        <pre>${JSON.stringify(diagnosis.resources, null, 2)}</pre>
    ` : ''}
</body>
</html>`;
}

export function deactivate() {}

class ContainerTreeProvider implements vscode.TreeDataProvider<ContainerItem> {
    private _onDidChangeTreeData = new vscode.EventEmitter<ContainerItem | undefined>();
    readonly onDidChangeTreeData = this._onDidChangeTreeData.event;
    private containers: any[] = [];

    constructor(private stoppedOnly: boolean) {
        this.refresh();
    }

    refresh() {
        const args = ['ps', '-a', '--format', '{{.Names}}\t{{.Status}}\t{{.Image}}'];
        cp.execFile('docker', args, (error, stdout) => {
            if (!error) {
                this.containers = stdout.trim().split('\n').filter(line => line).map(line => {
                    const [name, status, image] = line.split('\t');
                    return { name, status, image };
                }).filter(c => {
                    if (this.stoppedOnly) {
                        return !c.status.toLowerCase().includes('up');
                    }
                    return c.status.toLowerCase().includes('up');
                });
                this._onDidChangeTreeData.fire(undefined);
            }
        });
    }

    getTreeItem(element: ContainerItem): vscode.TreeItem {
        return element;
    }

    getChildren(): ContainerItem[] {
        return this.containers.map(c => new ContainerItem(
            c.name,
            c.status,
            c.image,
            vscode.TreeItemCollapsibleState.None
        ));
    }
}

class ContainerItem extends vscode.TreeItem {
    constructor(
        public readonly containerName: string,
        public readonly status: string,
        public readonly image: string,
        public readonly collapsibleState: vscode.TreeItemCollapsibleState
    ) {
        super(containerName, collapsibleState);
        this.tooltip = `${containerName}\n${status}\n${image}`;
        this.description = status;
        this.contextValue = 'container';
        this.iconPath = new vscode.ThemeIcon('debug-console');
    }
}
