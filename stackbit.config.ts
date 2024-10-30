import { defineStackbitConfig } from '@stackbit/types';
import {GitContentSource} from "@stackbit/cms-git";

export default defineStackbitConfig({
    "stackbitVersion": "~0.6.0",
    "nodeVersion": "18",
    "ssgName": "custom",
    "contentSources": [
        new GitContentSource({
            rootPath: "/Users/calin/Documents/LabelNet/GO Testing", contentDirs: [], models: []
        })
    ],
    "postInstallCommand": "npm i --no-save @stackbit/types"
})