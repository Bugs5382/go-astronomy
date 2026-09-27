import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

/**
 * Manual sidebar so the ordering of guides and reference pages is explicit
 * rather than filesystem-derived.
 */
const sidebars: SidebarsConfig = {
  docsSidebar: [
    'intro',
    'getting-started',
    'concepts',
    {
      type: 'category',
      label: 'Package reference',
      collapsed: false,
      items: [
        'reference/observer',
        'reference/sun',
        'reference/earth',
        'reference/moon',
        'reference/star',
        'reference/planet',
        'reference/constellation',
        'reference/project',
        'reference/errors',
      ],
    },
  ],
};

export default sidebars;
