import type { Application } from '../../../api/model';
import {
  ApplicationStoreProvider,
  useActiveTab,
  DetailTab,
} from '../stores/applicationStore';
import { WorkbenchHeader } from './WorkbenchHeader';
import { OverviewTab } from './tabs/OverviewTab';
import { CodeEditorTab } from './tabs/CodeEditorTab';
import { BuildsTab } from './tabs/BuildsTab';
import { TriggersTab } from './tabs/TriggersTab';
import { BindingsTab } from './tabs/BindingsTab';
import { SettingsTab } from './tabs/SettingsTab';

export interface ApplicationWorkbenchProps {
  app: Application;
  allApps?: Application[];
  initialTab?: DetailTab;
  onTabChange?: (tab: DetailTab) => void;
  onBack: () => void;
  onDeploy: (appId: string) => Promise<void> | void;
  isDeploying: boolean;
  onTestApp: (app: Application) => void;
  onRefreshApps?: () => void;
  onDeleteApp?: (appId: string) => Promise<void>;
  onNavigateToService?: (targetTab: string, resourceId?: string) => void;
}

export function ApplicationWorkbench(props: ApplicationWorkbenchProps) {
  const { app, initialTab } = props;

  return (
    <ApplicationStoreProvider
      key={app.id}
      appId={app.id}
      initialTab={initialTab || 'overview'}
      initialCode={app.inlineCode || ''}
    >
      <ApplicationWorkbenchContent {...props} />
    </ApplicationStoreProvider>
  );
}

function ApplicationWorkbenchContent({
  app,
  onBack,
  onDeploy,
  isDeploying,
  onTestApp,
  onTabChange,
  onDeleteApp,
  onRefreshApps,
  onNavigateToService,
}: ApplicationWorkbenchProps) {
  const activeTab = useActiveTab();

  return (
    <div className="flex-1 flex flex-col h-full overflow-hidden bg-zinc-950">
      <WorkbenchHeader
        app={app}
        onBack={onBack}
        onDeploy={onDeploy}
        isDeploying={isDeploying}
        onTestApp={onTestApp}
        onTabChange={onTabChange}
      />

      <div className="flex-1 overflow-y-auto">
        {activeTab === 'overview' && (
          <OverviewTab
            app={app}
            onTestApp={onTestApp}
            onNavigateToService={onNavigateToService}
          />
        )}
        {activeTab === 'code' && (
          <CodeEditorTab
            app={app}
            onDeploy={onDeploy}
            isDeploying={isDeploying}
            onRefreshApps={onRefreshApps}
          />
        )}
        {activeTab === 'builds' && (
          <BuildsTab
            app={app}
            onTestApp={onTestApp}
          />
        )}
        {activeTab === 'triggers' && (
          <TriggersTab
            app={app}
            onNavigateToService={onNavigateToService}
          />
        )}
        {activeTab === 'bindings' && (
          <BindingsTab
            app={app}
            onNavigateToService={onNavigateToService}
          />
        )}
        {activeTab === 'settings' && (
          <SettingsTab
            app={app}
            onDeleteApp={onDeleteApp}
            onRefreshApps={onRefreshApps}
          />
        )}
      </div>
    </div>
  );
}
