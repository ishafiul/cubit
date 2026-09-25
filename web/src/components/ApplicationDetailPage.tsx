import {
  ApplicationWorkbench,
  type ApplicationWorkbenchProps,
} from '../features/applications/components/ApplicationWorkbench';
import { getDeploymentStatusBadge } from '../features/applications/components/WorkbenchHeader';
import type { DetailTab } from '../features/applications/stores/applicationStore';
import {
  isServiceAttachedToApp,
  getServiceTabForBindingType,
  getServiceLabelForBindingType,
  getServicePageName,
} from '../features/applications/utils/binding-utils';

export {
  getDeploymentStatusBadge,
  isServiceAttachedToApp,
  getServiceTabForBindingType,
  getServiceLabelForBindingType,
  getServicePageName,
};

export type { DetailTab };

export interface ApplicationDetailPageProps extends ApplicationWorkbenchProps {}

export function ApplicationDetailPage(props: ApplicationDetailPageProps) {
  return <ApplicationWorkbench {...props} />;
}
