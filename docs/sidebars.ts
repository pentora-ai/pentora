import type { SidebarsConfig } from '@docusaurus/plugin-content-docs'

/**
 * Creating a sidebar enables you to:
 - create an ordered group of docs
 - render a sidebar for each doc of that group
 - provide next/previous navigation

 The sidebars can be generated from the filesystem, or explicitly defined here.

 Create as many sidebars as you want.
 */
const sidebars: SidebarsConfig = {
  // Main documentation sidebar
  docsSidebar: [
    {
      type: 'link',
      label: 'Home',
      href: '/',
      className: 'sidebar-nav-link sidebar-nav-home',
    },
    {
      type: 'link',
      label: 'Guides',
      href: '/getting-started/installation',
      className: 'sidebar-nav-link sidebar-nav-guides',
    },
    {
      type: 'link',
      label: 'CLI',
      href: '/cli/overview',
      className: 'sidebar-nav-link sidebar-nav-cli',
    },
    {
      type: 'link',
      label: 'API Reference',
      href: '/api/overview',
      className: 'sidebar-nav-link sidebar-nav-reference',
    },
    {
      type: 'html',
      value:
        '<hr style="margin: 0.75rem 0.75rem; border-color: rgba(156, 163, 175, 0.15);" />',
    },
    'intro',
    {
      type: 'category',
      label: 'Getting Started',
      items: [
        'getting-started/installation',
        'getting-started/quick-start',
        'getting-started/first-scan',
      ],
    },
    {
      type: 'category',
      label: 'Core Concepts',
      items: [
        'concepts/overview',
        'concepts/scan-pipeline',
        'concepts/workspace',
        'concepts/dag-engine',
        'concepts/modules',
        'concepts/fingerprinting',
      ],
    },
    {
      type: 'category',
      label: 'CLI Reference',
      items: [
        'cli/overview',
        'cli/scan',
        'cli/workspace',
        'cli/server',
        'cli/fingerprint',
      ],
    },
    {
      type: 'category',
      label: 'Configuration',
      items: [
        'configuration/overview',
        'configuration/scan-profiles',
        'configuration/workspace-config',
        'configuration/logging',
      ],
    },
    {
      type: 'category',
      label: 'Architecture',
      items: [
        'architecture/overview',
        'architecture/engine',
        'architecture/modules',
        'architecture/plugins',
        'architecture/data-flow',
      ],
    },
    {
      type: 'category',
      label: 'Advanced Features',
      items: [
        'advanced/custom-modules',
        'advanced/external-plugins',
        'advanced/hooks-events',
        'advanced/custom-fingerprints',
      ],
    },
    {
      type: 'category',
      label: 'Enterprise',
      items: [
        'enterprise/overview',
        'enterprise/licensing',
        'enterprise/distributed-scanning',
        'enterprise/multi-tenant',
        'enterprise/integrations',
      ],
    },
    {
      type: 'category',
      label: 'Deployment',
      items: [
        'deployment/standalone',
        'deployment/server-mode',
        'deployment/docker',
        'deployment/air-gapped',
      ],
    },
    {
      type: 'category',
      label: 'Guides',
      items: [
        'guides/network-scanning',
        'guides/vulnerability-assessment',
        'guides/compliance-checks',
        'guides/reporting',
      ],
    },
    {
      type: 'category',
      label: 'Troubleshooting',
      items: [
        'troubleshooting/common-issues',
        'troubleshooting/performance',
        'troubleshooting/debugging',
      ],
    },
  ],

  // API Reference sidebar
  apiSidebar: [
    {
      type: 'link',
      label: '📚 Documentation',
      href: '/intro',
      className: 'sidebar-main-link',
    },
    {
      type: 'link',
      label: '🔌 API Reference',
      href: '/api/overview',
      className: 'sidebar-main-link',
    },
    {
      type: 'html',
      value:
        '<hr style="margin: 1rem 0; border-color: var(--ifm-color-gray-300);" />',
    },
    'api/overview',
    {
      type: 'category',
      label: 'REST API',
      items: [
        'api/rest/authentication',
        'api/rest/scans',
        'api/rest/workspace',
        'api/rest/jobs',
      ],
    },
    {
      type: 'category',
      label: 'UI Portal',
      items: [
        'api/ui/overview',
        'api/ui/dashboard',
        'api/ui/scan-management',
        'api/ui/notifications',
      ],
    },
    {
      type: 'category',
      label: 'Module API',
      items: [
        'api/modules/interface',
        'api/modules/context',
        'api/modules/lifecycle',
      ],
    },
  ],
}

export default sidebars
