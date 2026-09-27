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
        {
          type: 'category',
          label: 'Planets',
          link: {type: 'doc', id: 'reference/planet'},
          items: [
            'reference/planets/mercury',
            'reference/planets/venus',
            'reference/planets/mars',
            'reference/planets/jupiter',
            'reference/planets/saturn',
            'reference/planets/uranus',
            'reference/planets/neptune',
          ],
        },
        'reference/constellation',
        'reference/project',
        'reference/errors',
      ],
    },
  ],
};

export default sidebars;
